package dashboards

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/slo"
)

const (
	rate      = "[$__rate_interval]"
	errorLogs = `detected_level=~"error|critical|fatal"`

	burnRateTicket = 1
	burnRatePage   = 6
	budgetLow      = 0.25
)

func sloSection(svc *service.Service) section {
	sloth := `{sloth_service=` + strconv.Quote(svc.Name) + `}`
	bySLO := func(series string) string { return "max by (sloth_slo) (" + series + sloth + ")" }

	return section{title: "SLOs e alertas", panels: []panel{
		&single{
			title:       "SLI atual",
			description: "Fração de eventos bons de cada SLO nos últimos 5 minutos, segundo as recording rules do Sloth.",
			unit:        "percentunit",
			width:       thirdWidth,
			queries:     []query{{expr: "1 - " + bySLO(catalog.SLOErrorRatioRate5m), legend: "{{sloth_slo}}"}},
		},
		&single{
			title: "Budget restante (30d)",
			description: "Fração do error budget de 30 dias que cada SLO ainda tem; abaixo de zero, o SLO já foi " +
				"descumprido na janela.",
			unit:    "percentunit",
			width:   thirdWidth,
			queries: []query{{expr: bySLO(catalog.SLOPeriodErrorBudgetRemainingRatio), legend: "{{sloth_slo}}"}},
			steps:   []step{{color: "red"}, at(0, "yellow"), at(budgetLow, "green")},
		},
		&single{
			title: "Burn rate",
			description: "Velocidade com que cada SLO queima o budget nos últimos 5 minutos: 1 gasta o budget " +
				"exatamente em 30 dias.",
			unit:    "short",
			width:   thirdWidth,
			queries: []query{{expr: bySLO(catalog.SLOCurrentBurnRateRatio), legend: "{{sloth_slo}}"}},
			steps:   []step{{color: "green"}, at(burnRateTicket, "yellow"), at(burnRatePage, "red")},
		},
		serviceAlerts{
			title:       "Alertas do serviço",
			description: "Alertas de cada processo de " + svc.Name + ", inclusive os de burn rate dos SLOs.",
			filter:      `{service=~` + strconv.Quote(svc.Name+"-.*") + `}`,
		},
	}}
}

func quantile(q, histogram, selector, by string) string {
	return "histogram_quantile(" + q + ", sum by (" + by + ") (rate(" + histogram + "_bucket{" + selector + "}" + rate + ")))"
}

func quantiles(histogram, selector string) []query {
	return []query{
		{expr: quantile("0.5", histogram, selector, "le"), legend: "p50", exemplar: true},
		{expr: quantile("0.99", histogram, selector, "le"), legend: "p99", exemplar: true},
	}
}

func seconds(value float64) *float64 { return &value }

func apiSection(svc *service.Service, processes []service.Process) section {
	if svc.Runtime == service.Node {
		return spanMetricsAPISection(processes)
	}

	selector := job(processes) + `, http_route!~` + strconv.Quote(slo.ProbeRoutes)
	requests := func(extra, by string) string {
		matchers := selector
		if extra != "" {
			matchers += ", " + extra
		}

		return "sum" + by + " (rate(" + catalog.HTTPServerRequestTotal + "{" + matchers + "}" + rate + "))"
	}

	return section{title: string(service.API), panels: []panel{
		&single{
			title:       "Taxa",
			description: "Requisições por segundo, fora as de health e readiness.",
			unit:        "reqps",
			width:       thirdWidth,
			queries:     []query{{expr: requests("", "")}},
		},
		&single{
			title:       "5xx",
			description: "Fração das requisições respondidas com 5xx, as que queimam o budget de api-availability.",
			unit:        "percentunit",
			width:       thirdWidth,
			queries: []query{{
				expr: "(" + requests(`http_response_status_code=~"5.."`, "") + " or vector(0)) / " + requests("", ""),
			}},
		},
		&single{
			title:       "p99",
			description: "Percentil 99 da duração das requisições no intervalo.",
			unit:        "s",
			width:       thirdWidth,
			queries:     []query{{expr: quantile("0.99", catalog.HTTPServerRequestDuration, selector, "le")}},
			steps:       []step{{color: "green"}, at(slo.APILatencyThresholdSeconds, "red")},
		},
		&graph{
			title:       "Taxa por rota",
			description: "Requisições por segundo por rota e método.",
			unit:        "reqps",
			width:       halfWidth,
			queries: []query{{
				expr:   requests("", " by (http_route, http_request_method)"),
				legend: "{{http_request_method}} {{http_route}}",
			}},
		},
		&graph{
			title:       "4xx/5xx por rota",
			description: "Respostas de erro por segundo, por rota e status; só o 5xx queima budget.",
			unit:        "reqps",
			width:       halfWidth,
			queries: []query{{
				expr:   requests(`http_response_status_code=~"4..|5.."`, " by (http_route, http_response_status_code)"),
				legend: "{{http_response_status_code}} {{http_route}}",
			}},
		},
		&graph{
			title: "Latência p50/p99",
			description: "Percentis 50 e 99 da duração das requisições; a linha é o limite do SLO api-latency. " +
				"Os pontos são exemplars: abrem o trace.",
			unit:    "s",
			width:   halfWidth,
			queries: quantiles(catalog.HTTPServerRequestDuration, selector),
			sloLine: seconds(slo.APILatencyThresholdSeconds),
		},
		&graph{
			title: "Latência p99 por rota",
			description: "Percentil 99 da duração por rota; a linha é o limite do SLO api-latency. Clique numa " +
				"série para ver os traces lentos da rota.",
			unit:  "s",
			width: halfWidth,
			queries: []query{{
				expr:     quantile("0.99", catalog.HTTPServerRequestDuration, selector, "le, http_route"),
				legend:   "{{http_route}}",
				exemplar: true,
			}},
			sloLine: seconds(slo.APILatencyThresholdSeconds),
			links:   []cog.Builder[dashboard.DashboardLink]{slowTracesLink(processes)},
		},
	}}
}

func consumerSection(_ *service.Service, processes []service.Process) section {
	selector := job(processes)
	groups := make([]string, 0, len(processes))

	for _, p := range processes {
		groups = append(groups, regexp.QuoteMeta(p.ConsumerGroup()))
	}

	consumerGroups := `{consumergroup=~` + strconv.Quote(strings.Join(groups, "|")) + `}`

	return section{title: string(service.Consumer), panels: []panel{
		&graph{
			title:       "Mensagens/s",
			description: "Mensagens processadas por segundo: com sucesso por tópico, e com falha por error_type.",
			unit:        "short",
			width:       halfWidth,
			queries: []query{
				{
					expr: "sum by (messaging_destination_name) (rate(" + catalog.MessagingClientConsumedMessages + "{" +
						selector + `, error_type=""}` + rate + "))",
					legend: "sucesso {{messaging_destination_name}}",
				},
				{
					expr: "sum by (error_type) (rate(" + catalog.MessagingClientConsumedMessages + "{" + selector +
						`, error_type!=""}` + rate + "))",
					legend: "falha {{error_type}}",
				},
			},
		},
		&graph{
			title: "Duração do processamento p50/p99",
			description: "Percentis 50 e 99 da duração de uma tentativa de processamento; a linha é o limite do " +
				"SLO consumer-latency. Os pontos são exemplars: abrem o trace.",
			unit:    "s",
			width:   halfWidth,
			queries: quantiles(catalog.MessagingProcessDuration, selector),
			sloLine: seconds(slo.ConsumerLatencyThresholdSeconds),
		},
		&graph{
			title:       "Retentativas",
			description: "Tentativas de processamento que falharam por segundo, por error_type; cada uma é retentada.",
			unit:        "short",
			width:       thirdWidth,
			queries: []query{{
				expr: "sum by (error_type) (rate(" + catalog.MessagingProcessDuration + "_count{" + selector +
					`, error_type!=""}` + rate + "))",
				legend: "{{error_type}}",
			}},
		},
		&graph{
			title:       "Lag do group",
			description: "Mensagens ainda não confirmadas pelo consumer group, por tópico; o alerta passa de 100.",
			unit:        "short",
			width:       thirdWidth,
			queries: []query{{
				expr:   "sum by (consumergroup, topic) (" + catalog.KafkaConsumerGroupUncommittedOffsets + consumerGroups + ")",
				legend: "{{consumergroup}} {{topic}}",
			}},
		},
		&graph{
			title:       "Envios à DLQ por causa",
			description: "Mensagens mandadas à DLQ no intervalo, por error_type: falharam de forma permanente.",
			unit:        "short",
			width:       thirdWidth,
			queries: []query{{
				expr: "sum by (error_type) (increase(" + catalog.MessagingDLQSentMessages + "{" + selector + "}" +
					rate + "))",
				legend: "{{error_type}}",
			}},
		},
	}}
}

func workerSection(svc *service.Service, processes []service.Process) section {
	selector := job(processes)
	age := catalog.OutboxOldestPendingAge(svc.Name) + "{" + selector + "}"

	return section{title: string(service.Worker), panels: []panel{
		&single{
			title:       "Pendente mais antiga",
			description: "Idade da mensagem mais antiga do outbox ainda não levada ao Kafka; 0 sem pendentes.",
			unit:        "s",
			width:       halfWidth,
			queries:     []query{{expr: "max(" + age + ")"}},
			steps:       []step{{color: "green"}, at(slo.OutboxFreshnessThresholdSeconds, "red")},
		},
		&single{
			title:       "Réplicas reportando",
			description: "Réplicas do worker que reportam a idade do outbox; 0 significa que ninguém esvazia o outbox.",
			unit:        "short",
			width:       halfWidth,
			queries:     []query{{expr: "count(" + age + ") or vector(0)"}},
			steps:       []step{{color: "red"}, at(1, "green")},
		},
		&graph{
			title: "Idade da pendente mais antiga",
			description: "Idade da mensagem pendente mais antiga do outbox, por processo; a linha é o limite do SLO " +
				"outbox-freshness.",
			unit:    "s",
			width:   halfWidth,
			queries: []query{{expr: "max by (job) (" + age + ")", legend: "{{job}}"}},
			sloLine: seconds(slo.OutboxFreshnessThresholdSeconds),
		},
		&graph{
			title:       "Envios/s",
			description: "Mensagens levadas ao Kafka por segundo: com sucesso por tópico, e com falha por error_type.",
			unit:        "short",
			width:       halfWidth,
			queries: []query{
				{
					expr: "sum by (messaging_destination_name) (rate(" + catalog.MessagingClientSentMessages + "{" +
						selector + `, error_type=""}` + rate + "))",
					legend: "sucesso {{messaging_destination_name}}",
				},
				{
					expr: "sum by (error_type) (rate(" + catalog.MessagingClientSentMessages + "{" + selector +
						`, error_type!=""}` + rate + "))",
					legend: "falha {{error_type}}",
				},
			},
		},
		&graph{
			title:       "Duração do envio p50/p99",
			description: "Percentis 50 e 99 da duração do envio ao Kafka. Os pontos são exemplars: abrem o trace.",
			unit:        "s",
			width:       halfWidth,
			queries:     quantiles(catalog.MessagingClientOperationDuration, selector),
		},
		&graph{
			title:       "Limpeza",
			description: "Execuções da limpeza do outbox por resultado, e mensagens já encaminhadas que ela apagou.",
			unit:        "short",
			width:       halfWidth,
			queries: []query{
				{
					expr:   "sum by (outcome) (increase(" + catalog.OutboxPurgeRuns(svc.Name) + "{" + selector + "}" + rate + "))",
					legend: "execuções {{outcome}}",
				},
				{
					expr:   "sum(increase(" + catalog.OutboxPurgedMessages(svc.Name) + "{" + selector + "}" + rate + "))",
					legend: "apagadas",
				},
			},
		},
	}}
}

func runtimeSection() section {
	byJob := func(expr string) string { return "sum by (job) (" + expr + ")" }

	return section{title: "Runtime", collapsed: true, panels: []panel{
		&graph{
			title:       "CPU",
			description: "CPU usada por processo em user e system; 100% é um núcleo inteiro.",
			unit:        "percentunit",
			width:       halfWidth,
			queries: []query{{
				expr:   "sum by (job, cpu_mode) (rate(" + catalog.ProcessCPUTime + "{" + selectedJob + "}" + rate + "))",
				legend: "{{job}} {{cpu_mode}}",
			}},
		},
		&graph{
			title:       "RSS",
			description: "Memória residente (RSS) por processo, como o kernel a vê: inclui o que o runtime Go não conta.",
			unit:        "bytes",
			width:       halfWidth,
			queries:     []query{{expr: byJob(catalog.ProcessMemoryUsage + "{" + selectedJob + "}"), legend: "{{job}}"}},
		},
		&graph{
			title:       "Goroutines",
			description: "Goroutines vivas por processo; uma subida sem volta costuma ser vazamento.",
			unit:        "short",
			width:       thirdWidth,
			queries:     []query{{expr: byJob(catalog.GoGoroutineCount + "{" + selectedJob + "}"), legend: "{{job}}"}},
		},
		&graph{
			title:       "Memória Go",
			description: "Memória usada pelo runtime Go por processo: heap e demais (other) e stacks.",
			unit:        "bytes",
			width:       thirdWidth,
			queries: []query{{
				expr:   "sum by (job, go_memory_type) (" + catalog.GoMemoryUsed + "{" + selectedJob + "})",
				legend: "{{job}} {{go_memory_type}}",
			}},
		},
		&graph{
			title:       "GC",
			description: "Bytes alocados no heap por segundo, e o tamanho de heap que dispara o próximo GC.",
			unit:        "bytes",
			width:       thirdWidth,
			queries: []query{
				{
					expr:   byJob("rate(" + catalog.GoMemoryAllocated + "{" + selectedJob + "}" + rate + ")"),
					legend: "alocação/s {{job}}",
				},
				{expr: byJob(catalog.GoMemoryGCGoal + "{" + selectedJob + "}"), legend: "meta do GC {{job}}"},
			},
		},
	}}
}

func databaseSection() section {
	applications := `{application_name=~"$` + processVariable + `"}`

	return section{title: "Banco", collapsed: true, panels: []panel{
		&graph{
			title:       "Pool",
			description: "Conexões do pool pgx em uso e o máximo do pool, por processo; o alerta passa de 90%.",
			unit:        "short",
			width:       halfWidth,
			queries: []query{
				{
					expr:   "sum by (job) (" + catalog.PgxpoolAcquiredConnections + "{" + selectedJob + "})",
					legend: "em uso {{job}}",
				},
				{
					expr:   "sum by (job) (" + catalog.PgxpoolMaxConnections + "{" + selectedJob + "})",
					legend: "máximo {{job}}",
				},
			},
		},
		&graph{
			title:       "Conexões no servidor",
			description: "Conexões que o Postgres vê de cada processo (application_name), por estado.",
			unit:        "short",
			width:       halfWidth,
			queries: []query{{
				expr:   "sum by (application_name, state) (" + catalog.PgClientConnections + applications + ")",
				legend: "{{application_name}} {{state}}",
			}},
		},
	}}
}

func logsSection() section {
	stream := `{service_namespace=` + strconv.Quote(config.Namespace) + `, service_name=~"$` + processVariable + `"}`

	return section{title: "Logs", panels: []panel{
		&graph{
			title:       "Volume por nível",
			description: "Linhas de log por nível de cada processo selecionado.",
			unit:        "short",
			width:       fullWidth,
			logQL:       true,
			queries: []query{{
				expr:   "sum by (detected_level) (count_over_time(" + stream + " [$__auto]))",
				legend: "{{detected_level}}",
			}},
		},
		logLines{
			title:       "Logs de erro",
			description: "Logs com nível error, critical ou fatal de cada processo selecionado; cada um abre o trace.",
			expr:        stream + " | " + errorLogs,
		},
	}}
}

// The Drilldown apps read their filters from scenes' URL format, key|operator|value, where a | inside the value is
// written __gfp__. Logs Drilldown escapes a regex value, so it gets one link per process.
func drilldownLinks(svc *service.Service) []cog.Builder[dashboard.DashboardLink] {
	link := func(title, tooltip, path string) cog.Builder[dashboard.DashboardLink] {
		return dashboard.NewDashboardLinkBuilder(title).
			Type(dashboard.DashboardLinkTypeLink).
			Icon("external link").
			Tooltip(tooltip).
			Url(path).
			KeepTime(true)
	}

	var names []string

	for _, p := range svc.Processes {
		if p.LongRunning() {
			names = append(names, p.Name())
		}
	}

	filter := func(label, operator, value string) string {
		return url.QueryEscape(label+"|"+operator+"|") + url.QueryEscape(value)
	}

	out := make([]cog.Builder[dashboard.DashboardLink], 0, 1+2*len(names))
	out = append(out, link("Traces", "Traces de "+svc.Name+" no Traces Drilldown",
		"/a/grafana-exploretraces-app/explore?var-ds=${"+tempoDS.variable()+"}"+
			"&var-filters="+filter("resource.service.name", "=~", strings.Join(names, "__gfp__"))+
			"&var-groupBy=resource.service.name"))

	for _, name := range names {
		out = append(out, link("Logs "+name, "Logs de "+name+" no Logs Drilldown",
			"/a/grafana-lokiexplore-app/explore/service_name/"+url.PathEscape(name)+"/logs?var-ds=${"+lokiDS.variable()+"}"+
				"&var-filters="+filter("service_name", "=", name)))
	}

	if svc.Runtime != service.Go {
		return out
	}

	for _, name := range names {
		out = append(out, link("Profiles "+name, "Profile de CPU de "+name+" no Profiles Drilldown",
			"/a/grafana-pyroscope-app/explore?explorationType=flame-graph&var-serviceName="+url.QueryEscape(name)+
				"&var-profileMetricId="+url.QueryEscape(cpuProfile)))
	}

	return out
}

const cpuProfile = "process_cpu:cpu:nanoseconds:cpu:nanoseconds"

var placeholder = regexp.MustCompile(`\$\{[^}]+\}`)

// queryEscape percent-encodes a URL query value but keeps Grafana's ${...} placeholders, which it interpolates first.
func queryEscape(value string) string {
	var out strings.Builder

	last := 0
	for _, loc := range placeholder.FindAllStringIndex(value, -1) {
		out.WriteString(url.QueryEscape(value[last:loc[0]]))
		out.WriteString(value[loc[0]:loc[1]])
		last = loc[1]
	}

	out.WriteString(url.QueryEscape(value[last:]))

	return out.String()
}

func slowTracesLink(processes []service.Process) cog.Builder[dashboard.DashboardLink] {
	names := make([]string, 0, len(processes))
	for _, p := range processes {
		names = append(names, p.Name())
	}

	slow := time.Duration(slo.APILatencyThresholdSeconds * float64(time.Second))
	traceQL := `{resource.service.name=~` + strconv.Quote(strings.Join(names, "|")) +
		` && kind=server && span.http.route="${__field.labels.http_route}" && duration > ` + slow.String() + `}`

	tempo := "${" + tempoDS.variable() + "}"
	panes := map[string]any{"slow": map[string]any{
		"datasource": tempo,
		"queries": []map[string]any{{
			"refId":      "A",
			"datasource": map[string]string{"type": string(tempoDS), "uid": tempo},
			"queryType":  "traceql",
			"query":      traceQL,
		}},
		"range": map[string]string{"from": "${__from}", "to": "${__to}"},
	}}

	var encoded bytes.Buffer

	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(panes)

	return dashboard.NewDashboardLinkBuilder("Traces lentos desta rota").
		Url("/explore?schemaVersion=1&panes=" + queryEscape(strings.TrimSpace(encoded.String()))).
		TargetBlank(true)
}
