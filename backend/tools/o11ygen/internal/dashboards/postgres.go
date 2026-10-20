package dashboards

import (
	"strconv"

	"github.com/grafana/grafana-foundation-sdk/go/bargauge"
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/table"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/rules"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	onDatabase  = `{datname=~"$database"}`
	topQueries  = "10"
	topMetrics  = "20"
	byStatement = "datname, user, queryid"
)

func postgresDashboard() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder("Postgres — Conexões · Locks · Queries · Cardinalidade").
		Uid(config.Namespace+"-postgres").
		Tags([]string{"postgres", config.Namespace, "sre"}).
		Readonly().
		Tooltip(1).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{
			datasourceVariable(prometheusDS),
			labelVariable("database", "Database", catalog.PgDatabaseSizeBytes, "datname", ".+").
				Regex("/^(?!template[01]$).*/"),
			labelVariable("job", "Job (cardinalidade)", catalog.JobSeriesCount, "job", ".*"),
		}).
		Annotations(grafanaAnnotations()).
		WithRow(row(1, "Visão geral", 0)).
		WithPanel(alertList(2, "Alertas do Postgres e de cardinalidade",
			"PostgresDown, PostgresConnectionsNearMax e SeriesBudgetExceeded de todo processo.",
			alertsNamed(rules.AlertNames("postgres")), true).
			GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 1})).
		WithPanel(statPanel(3, "Postgres no ar", "pg_up do exporter do Alloy: 1 conectado, 0 fora; sem valor, o exporter parou.",
			"short", dashboard.GridPos{H: 8, W: 4, X: 12, Y: 1}, "max("+catalog.PgUp+")",
			[]dashboard.Threshold{{Color: "red"}, {Value: cog.ToPtr[float64](1), Color: "green"}})).
		WithPanel(statPanel(4, "Conexões em uso", "Conexões abertas sobre max_connections. O alerta dispara acima de 80% por 5 min.",
			"percentunit", dashboard.GridPos{H: 8, W: 4, X: 16, Y: 1},
			"sum("+catalog.PgStatActivityCount+") / max("+catalog.PgSettingsMaxConnections+")",
			[]dashboard.Threshold{{Color: "green"}, {Value: cog.ToPtr(0.6), Color: "yellow"}, {Value: cog.ToPtr(0.8), Color: "red"}})).
		WithPanel(statPanel(5, "Cache hit", "Leituras de bloco servidas pelo shared_buffers nas databases selecionadas.",
			"percentunit", dashboard.GridPos{H: 8, W: 4, X: 20, Y: 1}, cacheHit(""),
			[]dashboard.Threshold{{Color: "red"}, {Value: cog.ToPtr(0.9), Color: "yellow"}, {Value: cog.ToPtr(0.99), Color: "green"}})).
		WithRow(row(6, "Conexões", 9)).
		WithPanel(timeseriesPanel(7, "Conexões por database e usuário",
			"Conexões abertas em pg_stat_activity, por database e role.", "short",
			dashboard.GridPos{H: 8, W: 8, X: 0, Y: 10},
			promQuery("A", "sum by (datname, usename) ("+catalog.PgClientConnections+onDatabase+")", "{{usename}}@{{datname}}"))).
		WithPanel(timeseriesPanel(8, "Conexões por application_name e estado",
			"Cada pool pgx se identifica como <service.name>: um serviço com conexões demais aparece aqui.", "short",
			dashboard.GridPos{H: 8, W: 8, X: 8, Y: 10},
			promQuery("A", "sum by (application_name, state) ("+catalog.PgClientConnections+onDatabase+")",
				"{{application_name}} {{state}}"))).
		WithPanel(timeseriesPanel(9, "Transação mais longa",
			"Idade da transação aberta mais antiga por database e estado; idle in transaction alto prende locks.", "s",
			dashboard.GridPos{H: 8, W: 8, X: 16, Y: 10},
			promQuery("A", "max by (datname, state) ("+catalog.PgStatActivityMaxTxDuration+`{datname=~"$database", state=~"active|idle in transaction.*"})`,
				"{{datname}} {{state}}"))).
		WithRow(row(10, "Locks e transações", 18)).
		WithPanel(timeseriesPanel(11, "Locks por modo", "Locks em pg_locks por database e modo.", "short",
			dashboard.GridPos{H: 8, W: 8, X: 0, Y: 19},
			promQuery("A", "sum by (datname, mode) ("+catalog.PgLocksCount+onDatabase+")", "{{datname}} {{mode}}"))).
		WithPanel(timeseriesPanel(12, "Transações (tx/s)", "Commits e rollbacks por segundo, por database.", "ops",
			dashboard.GridPos{H: 8, W: 8, X: 8, Y: 19},
			promQuery("A", "sum by (datname) (rate("+catalog.PgStatDatabaseXactCommit+onDatabase+"[$__rate_interval]))",
				"commit {{datname}}"),
			promQuery("B", "sum by (datname) (rate("+catalog.PgStatDatabaseXactRollback+onDatabase+"[$__rate_interval]))",
				"rollback {{datname}}"))).
		WithPanel(timeseriesPanel(13, "Deadlocks", "Deadlocks detectados por database.", "short",
			dashboard.GridPos{H: 8, W: 8, X: 16, Y: 19},
			promQuery("A", "sum by (datname) (increase("+catalog.PgStatDatabaseDeadlocks+onDatabase+"[$__rate_interval]))",
				"{{datname}}"))).
		WithRow(row(14, "Tamanho e cache", 27)).
		WithPanel(timeseriesPanel(15, "Tamanho por database", "pg_database_size por database.", "bytes",
			dashboard.GridPos{H: 8, W: 12, X: 0, Y: 28},
			promQuery("A", "sum by (datname) ("+catalog.PgDatabaseSizeBytes+onDatabase+")", "{{datname}}"))).
		WithPanel(timeseriesPanel(16, "Cache hit por database",
			"Leituras de bloco servidas pelo shared_buffers, por database. Abaixo de 99% o working set não cabe na memória.",
			"percentunit", dashboard.GridPos{H: 8, W: 12, X: 12, Y: 28},
			promQuery("A", cacheHit("datname"), "{{datname}}")).
			Min(0).
			Max(1)).
		WithRow(row(17, "pg_stat_statements", 36)).
		WithPanel(statementsTable()).
		WithPanel(timeseriesPanel(19, "Tempo de execução por query (top "+topQueries+")",
			"Segundos de execução por segundo das queries que mais ocupam o banco.", "s",
			dashboard.GridPos{H: 8, W: 12, X: 0, Y: 47},
			promQuery("A", "topk("+topQueries+", "+statementRate(catalog.PgStatStatementsSeconds)+")",
				"{{user}}@{{datname}} {{queryid}}"))).
		WithPanel(timeseriesPanel(20, "Tempo médio por chamada (top "+topQueries+")",
			"Duração média de uma execução, das queries mais lentas.", "s",
			dashboard.GridPos{H: 8, W: 12, X: 12, Y: 47},
			promQuery("A", "topk("+topQueries+", "+statementRate(catalog.PgStatStatementsSeconds)+" / "+
				statementRate(catalog.PgStatStatementsCalls)+")", "{{user}}@{{datname}} {{queryid}}"))).
		WithRow(row(21, "Cardinalidade", 55)).
		WithPanel(topMetricsPanel()).
		WithPanel(timeseriesPanel(23, "Séries por job",
			"job:series:count por job. SeriesBudgetExceeded dispara acima do teto do serviço (padrão "+
				strconv.Itoa(service.DefaultSeriesBudget)+", a linha vermelha) por 15 min.", "short",
			dashboard.GridPos{H: 12, W: 12, X: 12, Y: 56},
			promQuery("A", "sum by (job) ("+catalog.JobSeriesCount+`{job=~"$job"})`, "{{job}}")).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Color: "transparent"},
					{Value: cog.ToPtr(float64(service.DefaultSeriesBudget)), Color: "red"}})).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().Mode(common.GraphThresholdsStyleModeLine)))
}

func labelVariable(name, label, metric, labelName, allValue string) *dashboard.QueryVariableBuilder {
	query := "label_values(" + metric + ", " + labelName + ")"

	return dashboard.NewQueryVariableBuilder(name).
		Label(label).
		Query(dashboard.StringOrMap{Map: map[string]any{"query": query, "refId": name + "-var"}}).
		Datasource(datasourceRef(prometheusDS)).
		Current(dashboard.VariableOption{
			Selected: cog.ToPtr(true),
			Text:     dashboard.StringOrArrayOfString{String: cog.ToPtr("All")},
			Value:    dashboard.StringOrArrayOfString{String: cog.ToPtr("$__all")},
		}).
		Multi(true).
		Refresh(2).
		Sort(1).
		IncludeAll(true).
		AllValue(allValue).
		Definition(query)
}

func cacheHit(by string) string {
	sum := "sum"
	if by != "" {
		sum = "sum by (" + by + ")"
	}

	hit := sum + " (rate(" + catalog.PgStatDatabaseBlksHit + onDatabase + "[$__rate_interval]))"
	read := sum + " (rate(" + catalog.PgStatDatabaseBlksRead + onDatabase + "[$__rate_interval]))"

	return hit + " / (" + hit + " + " + read + ")"
}

func statementRate(metric string) string {
	return "sum by (" + byStatement + ") (rate(" + metric + onDatabase + "[$__rate_interval]))"
}

func statementsInRange(metric string) string {
	return "sum by (" + byStatement + ") (increase(" + metric + onDatabase + "[$__range]))"
}

func statementsTable() cog.Builder[dashboard.Panel] {
	top := "topk(" + topQueries + ", " + statementsInRange(catalog.PgStatStatementsSeconds) + ")"
	inTop := " and on (" + byStatement + ") " + top

	return table.NewPanelBuilder().
		Id(18).
		Title("Queries que mais ocuparam o banco no intervalo (top "+topQueries+")").
		Description("Tempo total, chamadas e linhas por query no intervalo do dashboard. "+
			"O texto da query: SELECT query FROM pg_stat_statements WHERE queryid = <queryid>.").
		Datasource(datasourceRef(prometheusDS)).
		Targets(promTargets(
			promInstantTable("A", top),
			promInstantTable("B", statementsInRange(catalog.PgStatStatementsCalls)+inTop),
			promInstantTable("C", statementsInRange(catalog.PgStatStatementsRows)+inTop),
		)).
		GridPos(dashboard.GridPos{H: 10, W: 24, X: 0, Y: 37}).
		Unit("s").
		Transformations([]dashboard.DataTransformerConfig{
			{Id: "merge", Options: map[string]any{}},
			{Id: "organize", Options: map[string]any{
				"excludeByName": map[string]bool{"Time": true},
				"renameByName": map[string]string{
					"datname": "Database", "user": "Usuário", "queryid": "queryid",
					"Value #A": "Tempo total", "Value #B": "Chamadas", "Value #C": "Linhas",
				},
			}},
			{Id: "sortBy", Options: map[string]any{"sort": []map[string]any{{"field": "Tempo total", "desc": true}}}},
		}).
		OverrideByName("Chamadas", []dashboard.DynamicConfigValue{{Id: "unit", Value: "short"}}).
		OverrideByName("Linhas", []dashboard.DynamicConfigValue{{Id: "unit", Value: "short"}})
}

func topMetricsPanel() cog.Builder[dashboard.Panel] {
	return bargauge.NewPanelBuilder().
		Id(22).
		Title("Top " + topMetrics + " métricas por número de séries").
		Description("Métricas com mais séries nos jobs selecionados: o topo é a origem provável de um aumento de cardinalidade.").
		Datasource(datasourceRef(prometheusDS)).
		Targets(promTargets(promQuery("A",
			"topk("+topMetrics+`, count by (__name__) ({__name__=~".+", job=~"$job"}))`, "{{__name__}}").Instant())).
		GridPos(dashboard.GridPos{H: 12, W: 12, X: 0, Y: 56}).
		Unit("short").
		Orientation(common.VizOrientationHorizontal).
		DisplayMode(common.BarGaugeDisplayModeBasic).
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "blue"}}))
}
