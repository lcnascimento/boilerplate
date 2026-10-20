package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/loki"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

func overviewDashboard() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder("Overview").
		Uid(config.Namespace+"-overview").
		Tags([]string{config.Namespace,
			"overview"}).
		Readonly().
		Tooltip(1).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{datasourceVariable(tempoDS),
			datasourceVariable(prometheusDS),
			datasourceVariable(lokiDS)}).
		Annotations([]cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
			Name("Annotations & Alerts").
			Datasource(common.DataSourceRef{Type: cog.ToPtr[string]("grafana"), Uid: cog.ToPtr[string]("-- Grafana --")}).
			Hide(true).
			IconColor("rgba(0, 211, 255, 1)").
			Type("dashboard").
			BuiltIn(1)}).
		Links([]cog.Builder[dashboard.DashboardLink]{dashboard.NewDashboardLinkBuilder("Dashboards").
			Type("dashboards").
			Icon("external link").
			Tags([]string{config.Namespace}).
			KeepTime(true)}).
		WithPanel(overviewPanel1()).
		WithPanel(overviewPanel2()).
		WithPanel(overviewPanel3()).
		WithPanel(overviewPanel4())
}

func overviewPanel1() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(1).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{http_route!~\"/health|/ping\"}[$__rate_interval]))").
			LegendFormat("{{job}}").
			RefId("A")}).
		Title("Requisições por Serviço (req/s)").
		Description("Taxa de requisições HTTP por serviço, sem /ping e /health.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 9, W: 12, X: 0, Y: 0}).
		Unit("reqps").
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("palette-classic")).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("table").
			Placement("bottom").
			ShowLegend(true).
			Calcs([]string{"mean",
				"max"})).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("multi").
			Sort("desc")).
		DrawStyle("line").
		LineInterpolation("linear").
		FillOpacity(10).
		ShowPoints("never")
}

func overviewPanel2() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(2).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("100 * (sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{http_route!~\"/health|/ping\", http_response_status_code=~\"5..\"}[$__rate_interval])) or sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{http_route!~\"/health|/ping\"}[$__rate_interval])) * 0) / sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{http_route!~\"/health|/ping\"}[$__rate_interval]))").
			LegendFormat("{{job}}").
			RefId("A")}).
		Title("Taxa de Erro 5xx por Serviço (%)").
		Description("Percentual de respostas 5xx sobre o total de requisições, por serviço.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 9, W: 12, X: 12, Y: 0}).
		Unit("percent").
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("palette-classic")).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("table").
			Placement("bottom").
			ShowLegend(true).
			Calcs([]string{"mean",
				"max"})).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("multi").
			Sort("desc")).
		DrawStyle("line").
		LineInterpolation("linear").
		FillOpacity(10).
		ShowPoints("never")
}

func overviewPanel3() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(3).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("histogram_quantile(0.95, sum by (job, le) (rate(" + catalog.HTTPServerRequestDuration + "_bucket{http_route!~\"/health|/ping\"}[$__rate_interval])))").
			LegendFormat("{{job}}").
			RefId("A")}).
		Title("Latência p95 por Serviço").
		Description("Percentil 95 da duração das requisições HTTP, por serviço.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 9, W: 12, X: 0, Y: 9}).
		Unit("s").
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("palette-classic")).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("table").
			Placement("bottom").
			ShowLegend(true).
			Calcs([]string{"mean",
				"max"})).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("multi").
			Sort("desc")).
		DrawStyle("line").
		LineInterpolation("linear").
		FillOpacity(10).
		ShowPoints("never")
}

func overviewPanel4() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(4).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("sum by (service_name) (count_over_time({service_name=~\".+\"} | detected_level=~\"error|critical|fatal\" [$__auto]))").
			LegendFormat("{{service_name}}").
			RefId("A").
			Datasource(datasourceRef(lokiDS))}).
		Title("Logs de Erro por Serviço").
		Description("Volume de linhas de log com error, fatal ou panic, por service_name (apps via OTLP e containers via docker).").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 9, W: 12, X: 12, Y: 9}).
		Unit("short").
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("palette-classic")).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("table").
			Placement("bottom").
			ShowLegend(true).
			Calcs([]string{"mean",
				"max"})).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("multi").
			Sort("desc")).
		DrawStyle("line").
		LineInterpolation("linear").
		FillOpacity(10).
		ShowPoints("never")
}
