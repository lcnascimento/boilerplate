package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/logs"
	"github.com/grafana/grafana-foundation-sdk/go/loki"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

func logsDashboard() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder("Logs Explorer").
		Uid(config.Namespace+"-logs").
		Tags([]string{config.Namespace,
			"logs"}).
		Readonly().
		Tooltip(0).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{datasourceVariable(lokiDS),
			dashboard.NewQueryVariableBuilder("service").
				Label("Service").
				Hide(0).
				Query(dashboard.StringOrMap{String: cog.ToPtr[string]("label_values(service_name)")}).
				Datasource(datasourceRef(lokiDS)).
				Current(dashboard.VariableOption{Text: dashboard.StringOrArrayOfString{}, Value: dashboard.StringOrArrayOfString{}}).
				Multi(true).
				Refresh(2).
				Sort(1).
				IncludeAll(true).
				AllValue(".+").
				Definition("label_values(service_name)"),
			dashboard.NewQueryVariableBuilder("container").
				Label("Container").
				Hide(0).
				Query(dashboard.StringOrMap{String: cog.ToPtr[string]("label_values(container)")}).
				Datasource(datasourceRef(lokiDS)).
				Current(dashboard.VariableOption{Text: dashboard.StringOrArrayOfString{}, Value: dashboard.StringOrArrayOfString{}}).
				Multi(true).
				Refresh(2).
				Sort(1).
				IncludeAll(true).
				AllValue(".*").
				Definition("label_values(container)")}).
		Annotations([]cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
			Name("Annotations & Alerts").
			Datasource(common.DataSourceRef{Type: cog.ToPtr[string]("grafana"), Uid: cog.ToPtr[string]("-- Grafana --")}).
			Hide(true).
			IconColor("rgba(0, 211, 255, 1)").
			Type("dashboard").
			BuiltIn(1)}).
		WithPanel(logsPanel1()).
		WithPanel(logsPanel2()).
		WithPanel(logsPanel3()).
		WithPanel(logsPanel4()).
		WithPanel(logsPanel5()).
		WithPanel(logsPanel6()).
		WithPanel(logsPanel7())
}

func logsPanel1() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(1).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("sum(count_over_time({service_name=~\"${service:regex}\", container=~\"${container:regex}\"} [$__auto]))").
			Instant(true).
			RefId("A").
			QueryType("instant")}).
		Title("Total Logs").
		Description("Linhas de log no intervalo selecionado.").
		Unit("short").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 4, W: 8, X: 0, Y: 0}).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "blue"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("thresholds")).
		GraphMode("none").
		ColorMode("value").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Values(false).
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func logsPanel2() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(2).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("sum(count_over_time({service_name=~\"${service:regex}\", container=~\"${container:regex}\"} | detected_level=\"error\" [$__auto]))").
			Instant(true).
			RefId("A").
			QueryType("instant")}).
		Title("Total Errors").
		Description("Linhas de log com nível error no intervalo selecionado.").
		Unit("short").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 4, W: 8, X: 8, Y: 0}).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](10), Color: "yellow"},
				{Value: cog.ToPtr[float64](50), Color: "red"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("thresholds")).
		GraphMode("none").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Values(false).
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func logsPanel3() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(3).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("sum(count_over_time({service_name=~\"${service:regex}\", container=~\"${container:regex}\"} | detected_level=\"warn\" [$__auto]))").
			Instant(true).
			RefId("A").
			QueryType("instant")}).
		Title("Total Warnings").
		Description("Linhas de log com nível warn no intervalo selecionado.").
		Unit("short").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 4, W: 8, X: 16, Y: 0}).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](20), Color: "yellow"},
				{Value: cog.ToPtr[float64](100), Color: "orange"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("thresholds")).
		GraphMode("none").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Values(false).
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func logsPanel4() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(4).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("sum by (service_name) (count_over_time({service_name=~\"${service:regex}\", container=~\"${container:regex}\"} [$__auto]))").
			LegendFormat("{{service_name}}").
			RefId("A")}).
		Title("Log Volume by Service").
		Description("Linhas de log por service_name ao longo do tempo.").
		Unit("short").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 4}).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("palette-classic")).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("table").
			Placement("bottom").
			ShowLegend(true).
			Calcs([]string{"sum"})).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("multi").
			Sort("desc")).
		DrawStyle("line").
		LineInterpolation("linear").
		FillOpacity(10).
		ShowPoints("never")
}

func logsPanel5() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(5).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("sum by (service_name) (count_over_time({service_name=~\"${service:regex}\", container=~\"${container:regex}\"} | detected_level=~\"error|critical|fatal\" [$__auto]))").
			LegendFormat("{{service_name}}").
			RefId("A")}).
		Title("Errors by Service").
		Description("Linhas de log com error, critical ou fatal, por service_name, ao longo do tempo.").
		Unit("short").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 4}).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("palette-classic")).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("table").
			Placement("bottom").
			ShowLegend(true).
			Calcs([]string{"sum"})).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("multi").
			Sort("desc")).
		DrawStyle("line").
		LineInterpolation("linear").
		FillOpacity(10).
		ShowPoints("never")
}

func logsPanel6() cog.Builder[dashboard.Panel] {
	return logs.NewPanelBuilder().
		Id(6).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("{service_name=~\"${service:regex}\", container=~\"${container:regex}\"}").
			RefId("A")}).
		Title("All Logs").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 12, W: 24, X: 0, Y: 12}).
		ShowLabels(true).
		ShowCommonLabels(false).
		ShowTime(true).
		ShowLogContextToggle(false).
		WrapLogMessage(false).
		PrettifyLogMessage(false).
		EnableLogDetails(true).
		SortOrder("Descending").
		DedupStrategy("none")
}

func logsPanel7() cog.Builder[dashboard.Panel] {
	return logs.NewPanelBuilder().
		Id(7).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Expr("{service_name=~\"${service:regex}\", container=~\"${container:regex}\"} | detected_level=~\"error|critical|fatal\"").
			RefId("A")}).
		Title("Error Logs").
		Datasource(datasourceRef(lokiDS)).
		GridPos(dashboard.GridPos{H: 12, W: 24, X: 0, Y: 24}).
		ShowLabels(true).
		ShowCommonLabels(false).
		ShowTime(true).
		ShowLogContextToggle(false).
		WrapLogMessage(false).
		PrettifyLogMessage(false).
		EnableLogDetails(true).
		SortOrder("Descending").
		DedupStrategy("none")
}
