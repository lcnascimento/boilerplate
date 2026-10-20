package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/logs"
	"github.com/grafana/grafana-foundation-sdk/go/loki"
	"github.com/grafana/grafana-foundation-sdk/go/nodegraph"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/tempo"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

func tracesDashboard() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder("Traces & Latência (APM)").
		Uid(config.Namespace+"-traces").
		Tags([]string{"traces",
			"apm",
			"tempo",
			config.Namespace}).
		Editable().
		Tooltip(0).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{datasourceVariable(lokiDS),
			datasourceVariable(tempoDS)}).
		Annotations([]cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
			Name("Annotations & Alerts").
			Datasource(common.DataSourceRef{Type: cog.ToPtr[string]("grafana"), Uid: cog.ToPtr[string]("-- Grafana --")}).
			Hide(true).
			IconColor("rgba(0, 211, 255, 1)").
			Type("dashboard").
			BuiltIn(1)}).
		WithPanel(tracesPanel1()).
		WithPanel(tracesPanel2()).
		WithPanel(tracesPanel3()).
		WithPanel(tracesPanel4()).
		WithPanel(tracesPanel5()).
		WithPanel(tracesPanel6())
}

func tracesPanel1() cog.Builder[dashboard.Panel] {
	return nodegraph.NewPanelBuilder().
		Id(1).
		Targets([]cog.Builder[variants.Dataquery]{tempo.NewDataqueryBuilder().
			RefId("A").
			QueryType("serviceMap").
			Datasource(datasourceRef(tempoDS))}).
		Title("Service Map").
		Datasource(datasourceRef(tempoDS)).
		GridPos(dashboard.GridPos{H: 10, W: 12, X: 0, Y: 0})
}

func tracesPanel2() cog.Builder[dashboard.Panel] {
	return dashboard.NewPanelBuilder().
		Type("traces").
		Id(2).
		Title("Traces Recentes").
		Datasource(datasourceRef(tempoDS)).
		GridPos(dashboard.GridPos{H: 10, W: 12, X: 12, Y: 0}).
		Targets([]cog.Builder[variants.Dataquery]{tempo.NewDataqueryBuilder().
			Datasource(datasourceRef(tempoDS)).
			Limit(20).
			Query("{}").
			QueryType("traceql").
			RefId("A")}).
		Options(map[string]any{
			"dedupStrategy":    "none",
			"enableLogDetails": true,
		})
}

func tracesPanel3() cog.Builder[dashboard.Panel] {
	return logs.NewPanelBuilder().
		Id(3).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("{service_name=~\".+\"} | detected_level=~\"error|critical|fatal\"").
			RefId("A").
			Datasource(datasourceRef(lokiDS))}).
		Title("Logs de Erro (APM)").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 10, W: 24, X: 0, Y: 10}).
		ShowLabels(true).
		ShowCommonLabels(false).
		ShowTime(false).
		ShowLogContextToggle(false).
		WrapLogMessage(true).
		PrettifyLogMessage(false).
		EnableLogDetails(true).
		SortOrder("Descending").
		DedupStrategy("none")
}

func tracesPanel4() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(4).
		Targets([]cog.Builder[variants.Dataquery]{tempo.NewDataqueryBuilder().
			RefId("A").
			QueryType("traceql").
			Query("{}").
			Limit(100).
			Datasource(datasourceRef(tempoDS))}).
		Title("Traces na Última Hora").
		Datasource(datasourceRef(tempoDS)).
		GridPos(dashboard.GridPos{H: 4, W: 8, X: 0, Y: 20}).
		Unit("short").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "blue"}})).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"count"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func tracesPanel5() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(5).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("count(sum by (service_name) (count_over_time({service_name=~\".+\"} | detected_level=~\"error|critical|fatal\" [1h])))").
			RefId("A").
			QueryType("instant").
			Datasource(datasourceRef(lokiDS))}).
		Title("Serviços com Erros (1h)").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 4, W: 8, X: 8, Y: 20}).
		Unit("short").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](1), Color: "yellow"},
				{Value: cog.ToPtr[float64](3), Color: "red"}})).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func tracesPanel6() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(6).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("sum by (service_name) (rate({service_name=~\".+\"} | detected_level=~\"error|critical|fatal\" [5m]))").
			LegendFormat("{{service_name}}").
			RefId("A").
			Datasource(datasourceRef(lokiDS))}).
		Title("Volume de Erros por Serviço (5m)").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 4, W: 8, X: 16, Y: 20}).
		Unit("short").
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none")).
		LineWidth(1).
		FillOpacity(20)
}
