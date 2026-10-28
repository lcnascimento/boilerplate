package dashboards

import (
	"slices"
	"strconv"
	"strings"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

// Next.js emits no http_server_* series, only a SERVER span per request named after its route.
func spanMetricsAPISection(_ *service.Service, processes []service.Process) section {
	selector := `service=~` + strconv.Quote(strings.Join(processNames(processes), "|")) + `, span_kind="SPAN_KIND_SERVER", ` +
		selectedEnvironment
	failed := `status_code="STATUS_CODE_ERROR"`
	calls := func(extra, by string) string {
		matchers := selector
		if extra != "" {
			matchers += ", " + extra
		}

		return "sum" + by + " (rate(" + catalog.TracesSpanMetricsCalls + "{" + matchers + "}" + rate + "))"
	}

	route := groupBy("http_status_code", "span_name")

	return section{title: string(service.API), panels: slices.Concat([]panel{
		&single{
			title:       "Taxa",
			description: "Requisições por segundo que o servidor atendeu.",
			unit:        "reqps",
			width:       thirdWidth,
			queries:     []query{{expr: calls("", "")}},
		},
		&single{
			title:       "Erros",
			description: "Fração das requisições cujo span terminou com erro: render, Server Action ou route handler que falhou.",
			unit:        "percentunit",
			width:       thirdWidth,
			queries:     []query{{expr: "(" + calls(failed, "") + " or vector(0)) / " + calls("", "")}},
		},
		&single{
			title:       "p99",
			description: "Percentil 99 da duração das requisições no intervalo.",
			unit:        "s",
			width:       thirdWidth,
			queries:     []query{{expr: quantile("0.99", catalog.TracesSpanMetricsLatency, selector, "le")}},
		},
		&graph{
			title:       "Taxa por rota",
			description: "Requisições por segundo por status, método e rota (o nome do span).",
			unit:        "reqps",
			width:       halfWidth,
			queries:     []query{{expr: calls("", " by ("+route.by+")"), legend: route.legend}},
		},
		&graph{
			title:       "Erros por rota",
			description: "Requisições com erro por segundo, por status, método e rota.",
			unit:        "reqps",
			width:       halfWidth,
			queries:     []query{{expr: calls(failed, " by ("+route.by+")"), legend: route.legend}},
		},
	}, (&latency{
		title:       "Latência",
		description: "da duração das requisições.",
		histogram:   catalog.TracesSpanMetricsLatency,
		selector:    selector,
	}).panels(), (&latency{
		title:       "Latência",
		qualifier:   " por rota",
		description: "da duração por status, método e rota.",
		histogram:   catalog.TracesSpanMetricsLatency,
		selector:    selector,
		grouping:    route,
	}).panels())}
}

func nodeRuntimeSection() section {
	selected := "{" + selectedJob + "}"

	return section{title: "Runtime", collapsed: true, panels: []panel{
		&graph{
			title:       "Event loop",
			description: "Percentil 99 do atraso do event loop por processo: o tempo que um callback pronto espera para rodar.",
			unit:        "s",
			width:       halfWidth,
			queries: []query{{
				expr: "max by (job) (" + catalog.NodejsEventloopDelayP99 + selected + ")", legend: "{{job}}",
			}},
		},
		&graph{
			title:       "Utilização do event loop",
			description: "Fração do tempo em que o event loop esteve ocupado; perto de 1, o processo está saturado.",
			unit:        "percentunit",
			width:       halfWidth,
			queries: []query{{
				expr: "max by (job) (" + catalog.NodejsEventloopUtilization + selected + ")", legend: "{{job}}",
			}},
		},
		&graph{
			title:       "Heap",
			description: "Heap do V8 em uso por processo; uma subida sem volta costuma ser vazamento.",
			unit:        "bytes",
			width:       halfWidth,
			queries: []query{{
				expr: "sum by (job) (" + catalog.V8jsMemoryHeapUsed + selected + ")", legend: "{{job}}",
			}},
		},
		&graph{
			title:       "GC",
			description: "Segundos gastos em GC por segundo, por processo e tipo de coleta.",
			unit:        "s",
			width:       halfWidth,
			queries: []query{{
				expr:   "sum by (job, v8js_gc_type) (rate(" + catalog.V8jsGCDuration + "_sum" + selected + rate + "))",
				legend: "{{job}} {{v8js_gc_type}}",
			}},
		},
	}}
}
