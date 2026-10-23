package rules

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/slo"
)

const (
	overviewDashboard = "overview"
	kafkaDashboard    = "kafka"
	postgresDashboard = "postgres"

	sustainedFor = 5 * time.Minute

	IdentityChangeBacklogThreshold = 5 * time.Minute

	seriesBudgetSustainedFor = 15 * time.Minute

	unauthenticatedRatio = "0.1"

	postgresDown     = "PostgresDown"
	kafkaBrokersDown = "KafkaBrokersDown"
)

type process struct {
	service.Process

	service *service.Service
}

func (p *process) job() string {
	return `{job=` + strconv.Quote(config.Namespace+"/"+p.Name()) + `}`
}

type healthDependency struct {
	name      string
	downAlert string
	checkedBy func(*process) bool
}

var healthDependencies = []healthDependency{
	{name: "o Postgres", downAlert: postgresDown, checkedBy: func(p *process) bool { return p.DBPool() }},
	{name: "o consumer", checkedBy: func(p *process) bool { return p.Runs(service.Consumer) }},
	{name: "o forwarder", checkedBy: func(p *process) bool { return p.Runs(service.Worker) }},
	{name: "o Kafka", downAlert: kafkaBrokersDown, checkedBy: func(p *process) bool {
		return p.Runs(service.Consumer) || p.Runs(service.Worker)
	}},
}

func (p *process) healthChecks() (names, downAlerts []string) {
	for _, dependency := range healthDependencies {
		if !dependency.checkedBy(p) {
			continue
		}

		names = append(names, dependency.name)
		if dependency.downAlert != "" {
			downAlerts = append(downAlerts, dependency.downAlert)
		}
	}

	return names, downAlerts
}

func (p *process) unauthenticatedRatio() string {
	q := slo.Unauthenticated(p.service.Runtime, []service.Process{p.Process})

	return fmt.Sprintf("sum(rate(%s{%s, %s}[5m])) / sum(rate(%s{%s}[5m]))", q.Series, q.Selector, q.Matcher, q.Series, q.Selector)
}

func enumerated(items []string) string {
	if len(items) == 1 {
		return items[0]
	}

	return strings.Join(items[:len(items)-1], ", ") + " e " + items[len(items)-1]
}

type inheritedAlert struct {
	inheritedBy func(*process) bool
	alert       func(*process) alert
}

var processAlerts = []inheritedAlert{
	{
		inheritedBy: func(p *process) bool { return p.LongRunning() },
		alert: func(p *process) alert {
			return alert{
				name:     "ServiceTelemetryAbsent",
				expr:     fmt.Sprintf("absent_over_time(%s%s[5m])", catalog.TargetInfo, p.job()),
				severity: critical,
				summary:  p.Name() + " não envia telemetria há 5 minutos",
				description: "Nenhum target_info de " + p.Name() + " chegou ao Prometheus nos últimos 5 minutos: " +
					"o processo caiu, travou ou não alcança o Alloy. Veja primeiro se ele está no ar e os logs dele.",
				dashboard: p.service.Name,
			}
		},
	},
	{
		inheritedBy: func(p *process) bool {
			checked, _ := p.healthChecks()

			return p.LongRunning() && p.service.Runtime.RunsGoKit() && len(checked) > 0
		},
		alert: func(p *process) alert {
			checked, downAlerts := p.healthChecks()
			checks := func(status string) string {
				return fmt.Sprintf(`sum by (job) (increase(%s{job=%s, http_route="/health", http_response_status_code=~%q}[2m])) > 0`,
					catalog.HTTPServerRequestTotal, strconv.Quote(config.Namespace+"/"+p.Name()), status)
			}

			return alert{
				name:     "HealthCheckFailing",
				expr:     checks("5..") + " unless " + checks("2.."),
				severity: critical,
				summary:  p.Name() + " responde 5xx em todo /health há 2 minutos",
				description: "Todo /health de " + p.Name() + " respondeu 5xx nos últimos 2 minutos: uma dependência que ele " +
					"checa (" + enumerated(checked) + ") está fora, e quem faz o probe o considera fora do ar. " +
					"Veja primeiro nos logs de " + p.Name() + " por que o health check falha e os alertas das dependências " +
					"(" + strings.Join(downAlerts, ", ") + ").",
				dashboard: p.service.Name,
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.DBPool() },
		alert: func(p *process) alert {
			return alert{
				name: "DBPoolSaturated",
				expr: fmt.Sprintf("max(%s%s / %s%s) > 0.9",
					catalog.PgxpoolAcquiredConnections, p.job(), catalog.PgxpoolMaxConnections, p.job()),
				forDuration: sustainedFor,
				severity:    warning,
				summary:     "Pool do Postgres de " + p.Name() + " com {{ $value | humanizePercentage }} das conexões em uso",
				description: "Mais de 90% das conexões do pool pgx de " + p.Name() + " estão em uso há 5 minutos, " +
					"então as queries esperam por conexão. Veja primeiro as queries lentas e os locks no Postgres, " +
					"depois o volume de requisições do processo.",
				dashboard: p.service.Name,
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.ActorAuth() },
		alert: func(p *process) alert {
			return alert{
				name:        "AuthenticationFailuresHigh",
				expr:        p.unauthenticatedRatio() + " > " + unauthenticatedRatio,
				forDuration: sustainedFor,
				severity:    warning,
				summary:     p.Name() + " responde 401 a {{ $value | humanizePercentage }} das requisições",
				description: "Mais de 10% das requisições de " + p.Name() + " respondem 401 há 5 minutos: " +
					"credenciais ou tokens recusados em massa. Veja primeiro os motivos na linha Autenticação e " +
					"autorização do dashboard do serviço.",
				dashboard: p.service.Name,
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.Runs(service.Consumer) },
		alert: func(p *process) alert {
			return alert{
				name: "ConsumerLagHigh",
				expr: fmt.Sprintf("sum by (consumergroup) (%s{consumergroup=%s}) > 100",
					catalog.KafkaConsumerGroupUncommittedOffsets, strconv.Quote(p.ConsumerGroup())),
				forDuration: sustainedFor,
				severity:    warning,
				summary: "Consumer group " + p.ConsumerGroup() + " de " + p.Name() +
					" com {{ $value | humanize }} mensagens atrasadas",
				description: "O consumer group " + p.ConsumerGroup() + " está mais de 100 mensagens atrás há 5 minutos: " +
					"o consumo está mais lento que a produção, ou parado. Veja primeiro se " + p.Name() +
					" está no ar e a duração do processamento na seção consumer do dashboard do serviço.",
				dashboard: p.service.Name,
				listedOn:  []string{kafkaDashboard},
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.Runs(service.Consumer) },
		alert: func(p *process) alert {
			return alert{
				name: "DLQMessageReceived",
				expr: fmt.Sprintf("sum by (topic) (increase(%s{topic=~%s}[5m])) > 0",
					catalog.KafkaTopicPartitionCurrentOffset, strconv.Quote(".+\\."+regexp.QuoteMeta(p.ConsumerGroup())+"\\.dlq")),
				severity: critical,
				summary:  "{{ $value | humanize }} mensagem(ns) nova(s) na DLQ {{ $labels.topic }} de " + p.Name(),
				description: p.Name() + " mandou mensagens à DLQ: elas falharam de forma permanente e não serão " +
					"reprocessadas sozinhas. Veja primeiro o error_type no painel de envios à DLQ do dashboard do serviço " +
					"e os logs de " + p.Name() + ".",
				dashboard: p.service.Name,
				listedOn:  []string{kafkaDashboard},
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.Runs(service.Worker) },
		alert: func(p *process) alert {
			gauge := catalog.OutboxOldestPendingAge(p.service.Name)

			return alert{
				name:        "OutboxAgeAbsent",
				expr:        fmt.Sprintf("absent(%s%s)", gauge, p.job()),
				forDuration: sustainedFor,
				severity:    critical,
				summary:     p.Name() + " não reporta a idade do outbox há 5 minutos",
				description: p.Name() + " não reporta " + gauge + ": sem forwarder ninguém esvazia o outbox e os " +
					"Integration Events param de sair. Veja primeiro se " + p.Name() + " está no ar e os logs dele.",
				dashboard: p.service.Name,
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.IdentityChanges() },
		alert: func(p *process) alert {
			return alert{
				name: "IdentityChangeBacklogHigh",
				expr: fmt.Sprintf("max(%s%s) > %d",
					catalog.IAMIdentityChangeOldestPendingAge, p.job(), int(IdentityChangeBacklogThreshold.Seconds())),
				severity: critical,
				summary:  p.Name() + " com a mudança de identidade mais antiga há {{ $value | humanizeDuration }} sem tradução",
				description: "A mudança de identidade pendente mais antiga de " + p.Name() + " passou de " +
					strconv.Itoa(int(IdentityChangeBacklogThreshold.Minutes())) + " minutos sem virar " +
					"Integration Event: a tradução parou, ou travou numa mudança que ela não sabe traduzir, e o Audit Log " +
					"deixa de receber as mudanças de identidade. Veja primeiro os logs de " + p.Name() +
					" (UNTRANSLATABLE_CHANGE) e a idade da mudança pendente mais antiga no dashboard do serviço.",
				dashboard: p.service.Name,
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.LongRunning() },
		alert: func(p *process) alert {
			budget := strconv.Itoa(p.service.EffectiveSeriesBudget())

			return alert{
				name:        "SeriesBudgetExceeded",
				expr:        fmt.Sprintf("%s%s > %s", catalog.JobSeriesCount, p.job(), budget),
				forDuration: seriesBudgetSustainedFor,
				severity:    warning,
				summary:     p.Name() + " com {{ $value | humanize }} séries, acima do teto de " + budget,
				description: p.Name() + " mantém mais de " + budget + " séries no Prometheus há 15 minutos: " +
					"algum atributo de métrica tem valores sem limite (ID, URL crua, mensagem de erro). " +
					"Veja primeiro quais métricas do job têm mais séries, depois o atributo que as multiplica.",
				dashboard: postgresDashboard,
			}
		},
	},
	{
		inheritedBy: func(p *process) bool { return p.LongRunning() },
		alert: func(p *process) alert {
			return alert{
				name:     "CriticalLogRecorded",
				expr:     recordsIn5m(catalog.ServiceLogRecordsCritical, p.Name(), nil) + " * 60 > 0",
				severity: critical,
				summary:  "{{ $value | humanize }} log(s) Critical ou Fatal de " + p.Name() + " nos últimos 5 minutos",
				description: p.Name() + " registrou um log com severidade Critical ou Fatal (severity_number >= 21), " +
					"o que costuma ser um panic ou um bug que não queima budget de SLO. Veja primeiro esses logs na " +
					"seção Logs do dashboard do serviço e o trace ligado a cada um.",
				dashboard: p.service.Name,
			}
		},
	},
}

func recordsIn5m(series, process string, extra map[string]string) string {
	matchers := `service=` + strconv.Quote(process)
	for _, name := range slices.Sorted(maps.Keys(extra)) {
		matchers += `, ` + name + `=` + strconv.Quote(extra[name])
	}

	return fmt.Sprintf("sum_over_time(%s{%s}[5m])", series, matchers)
}

func logAlert(p *process, declared *service.LogAlert) alert {
	return alert{
		name: declared.Name,
		expr: fmt.Sprintf("%s * 60 > %d", recordsIn5m(catalog.ServiceLogAlertRecords, p.Name(),
			map[string]string{"log_alert": declared.Name}), declared.MaxRecordsPer5m),
		severity:    severity(declared.Severity),
		summary:     declared.Summary,
		description: declared.Description,
		dashboard:   p.service.Name,
		service:     p.Name(),
	}
}

var recordingGroups = []group{{
	name: "cardinality",
	records: []recording{{
		name: catalog.JobSeriesCount,
		expr: `count by (job) ({__name__=~".+"})`,
	}},
}}

func infra(name string, alerts ...alert) group {
	for i := range alerts {
		alerts[i].service = name
	}

	return group{name: name, alerts: alerts}
}

// exporterDown fires on the exporter's own verdict, on a failed scrape (an exporter slower than the scrape timeout
// reports nothing, not its verdict), and once the exporter sends nothing at all.
func exporterDown(job, verdict string) string {
	up := catalog.Up + `{job=` + strconv.Quote(job) + `}`

	return fmt.Sprintf("%s or %s == 0 or absent_over_time(%s[5m])", verdict, up, up)
}

var infraGroups = []group{
	infra("kafka", alert{
		name:     kafkaBrokersDown,
		expr:     exporterDown(catalog.KafkaExporterJob, catalog.KafkaBrokers+" < 1"),
		severity: critical,
		summary:  "Kafka fora do alcance do exporter do Alloy, ou sem métricas dele há 5 minutos",
		description: "O exporter do Alloy não alcança nenhum broker do Kafka (a coleta falha ou vê zero brokers), " +
			"ou parou de mandar métricas: nenhum Integration Event é publicado nem consumido. " +
			"Veja primeiro o container kafka e depois o Alloy.",
		dashboard: kafkaDashboard,
	}),
	infra("postgres", alert{
		name:     postgresDown,
		expr:     exporterDown(catalog.PostgresExporterJob, catalog.PgUp+" == 0"),
		severity: critical,
		summary:  "Postgres fora do alcance do exporter do Alloy, ou sem métricas dele há 5 minutos",
		description: "O exporter do Alloy não consegue se conectar ao Postgres (pg_up 0 ou a coleta falha), " +
			"ou parou de mandar métricas: todo serviço com banco falha. Veja primeiro o container postgres e depois o Alloy.",
		dashboard: postgresDashboard,
	}, alert{
		name: "PostgresConnectionsNearMax",
		expr: fmt.Sprintf("sum(%s) / max(%s) > 0.8",
			catalog.PgStatActivityCount, catalog.PgSettingsMaxConnections),
		forDuration: sustainedFor,
		severity:    warning,
		summary:     "Postgres com {{ $value | humanizePercentage }} de max_connections em uso",
		description: "Mais de 80% de max_connections do Postgres está em uso há 5 minutos; perto do teto, " +
			"novas conexões são recusadas. Veja primeiro quais application_name têm mais conexões no dashboard postgres.",
		dashboard: postgresDashboard,
	}),
}
