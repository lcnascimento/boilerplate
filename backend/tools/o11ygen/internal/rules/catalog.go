package rules

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	overviewDashboard  = "overview"
	messagingDashboard = "messaging"
	logsDashboard      = "logs"

	sustainedFor = 5 * time.Minute

	seriesBudgetSustainedFor = 15 * time.Minute
)

type process struct {
	service.Process

	service service.Service
}

func (p *process) job() string {
	return `{job=` + strconv.Quote(config.Namespace+"/"+p.Name()) + `}`
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
				dashboard: overviewDashboard,
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
				dashboard: overviewDashboard,
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
					" está no ar e a duração do processamento no dashboard messaging.",
				dashboard: messagingDashboard,
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
					"reprocessadas sozinhas. Veja primeiro o error_type no painel de envios à DLQ do dashboard messaging " +
					"e os logs de " + p.Name() + ".",
				dashboard: messagingDashboard,
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
				dashboard: messagingDashboard,
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
				dashboard: overviewDashboard,
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
					"o que costuma ser um panic ou um bug que não queima budget de SLO. Veja primeiro esses logs no " +
					"dashboard logs e o trace ligado a cada um.",
				dashboard: logsDashboard,
			}
		},
	},
}

// recordsIn5m sums a Loki recorded rate1m over 5 minutes: the ruler writes one sample a minute, absent when no log
// matched, so increase() would miss a single record.
func recordsIn5m(series, process string, extra map[string]string) string {
	matchers := `service=` + strconv.Quote(process)
	for _, name := range slices.Sorted(maps.Keys(extra)) {
		matchers += `, ` + name + `=` + strconv.Quote(extra[name])
	}

	return fmt.Sprintf("sum_over_time(%s{%s}[5m])", series, matchers)
}

func logAlert(declared *service.LogAlert) alert {
	return alert{
		name: declared.Name,
		expr: fmt.Sprintf("%s * 60 > %d", recordsIn5m(catalog.ServiceLogAlertRecords, declared.Process,
			map[string]string{"log_alert": declared.Name}), declared.MaxRecordsPer5m),
		severity:    severity(declared.Severity),
		summary:     declared.Summary,
		description: declared.Description,
		dashboard:   logsDashboard,
		service:     declared.Process,
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

var infraGroups = []group{
	infra("kafka", alert{
		name:     "KafkaBrokersDown",
		expr:     fmt.Sprintf("%s < 1 or absent_over_time(%s[5m])", catalog.KafkaBrokers, catalog.KafkaBrokers),
		severity: critical,
		summary:  "Kafka sem brokers no ar, ou sem métricas do exporter há 5 minutos",
		description: "O exporter do Alloy não vê nenhum broker do Kafka, ou parou de mandar métricas: " +
			"nenhum Integration Event é publicado nem consumido. Veja primeiro o container kafka e depois o Alloy.",
		dashboard: messagingDashboard,
	}),
	infra("postgres", alert{
		name:     "PostgresDown",
		expr:     fmt.Sprintf("%s == 0 or absent_over_time(%s[5m])", catalog.PgUp, catalog.PgUp),
		severity: critical,
		summary:  "Postgres fora do ar, ou sem métricas do exporter há 5 minutos",
		description: "O exporter do Alloy não consegue se conectar ao Postgres, ou parou de mandar métricas: " +
			"todo serviço com banco falha. Veja primeiro o container postgres e depois o Alloy.",
		dashboard: overviewDashboard,
	}, alert{
		name: "PostgresConnectionsNearMax",
		expr: fmt.Sprintf("sum(%s) / max(%s) > 0.8",
			catalog.PgStatActivityCount, catalog.PgSettingsMaxConnections),
		forDuration: sustainedFor,
		severity:    warning,
		summary:     "Postgres com {{ $value | humanizePercentage }} de max_connections em uso",
		description: "Mais de 80% de max_connections do Postgres está em uso há 5 minutos; perto do teto, " +
			"novas conexões são recusadas. Veja primeiro quais application_name têm mais conexões em pg_stat_activity.",
		dashboard: overviewDashboard,
	}),
}
