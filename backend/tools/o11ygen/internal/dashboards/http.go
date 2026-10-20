package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

func httpDashboard() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder("HTTP APIs — Rate · Errors · Duration").
		Uid(config.Namespace+"-http").
		Tags([]string{"http",
			"red",
			config.Namespace,
			"sre"}).
		Editable().
		Tooltip(0).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{datasourceVariable(prometheusDS),
			dashboard.NewQueryVariableBuilder("service").
				Label("Serviço").
				Query(dashboard.StringOrMap{String: cog.ToPtr[string]("label_values(" + catalog.HTTPServerRequestTotal + ", job)")}).
				Datasource(datasourceRef(prometheusDS)).
				Current(dashboard.VariableOption{Selected: cog.ToPtr[bool](true), Text: dashboard.StringOrArrayOfString{String: cog.ToPtr[string]("All")}, Value: dashboard.StringOrArrayOfString{String: cog.ToPtr[string]("$__all")}}).
				Multi(true).
				Refresh(2).
				Sort(1).
				IncludeAll(true).
				AllValue(".+").
				Definition("label_values(" + catalog.HTTPServerRequestTotal + ", job)")}).
		Annotations([]cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
			Name("Annotations & Alerts").
			Datasource(common.DataSourceRef{Type: cog.ToPtr[string]("grafana"), Uid: cog.ToPtr[string]("-- Grafana --")}).
			Hide(true).
			IconColor("rgba(0, 211, 255, 1)").
			Type("dashboard").
			BuiltIn(1)}).
		WithRow(httpRow1()).
		WithPanel(httpPanel2()).
		WithPanel(httpPanel3()).
		WithPanel(httpPanel4()).
		WithPanel(httpPanel5()).
		WithRow(httpRow6()).
		WithPanel(httpPanel7()).
		WithPanel(httpPanel8()).
		WithRow(httpRow9()).
		WithPanel(httpPanel10()).
		WithPanel(httpPanel11()).
		WithRow(httpRow12()).
		WithPanel(httpPanel13()).
		WithPanel(httpPanel14())
}

func httpRow1() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Visão Geral").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 0}).
		Id(1)
}

func httpPanel2() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(2).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum(rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval]))").
			RefId("A")}).
		Title("Rate (req/s)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 0, Y: 1}).
		Unit("reqps").
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func httpPanel3() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(3).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("100 * (sum(rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\", http_response_status_code=~\"5..\"}[$__rate_interval])) or vector(0)) / sum(rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval]))").
			RefId("A")}).
		Title("Error Rate 5xx (%)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 6, Y: 1}).
		Unit("percent").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](1), Color: "yellow"},
				{Value: cog.ToPtr[float64](5), Color: "red"}})).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func httpPanel4() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(4).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("histogram_quantile(0.50, sum by (le) (rate(" + catalog.HTTPServerRequestDuration + "_bucket{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval])))").
			RefId("A")}).
		Title("P50 Latência").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 12, Y: 1}).
		Unit("s").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](0.5), Color: "yellow"},
				{Value: cog.ToPtr[float64](1), Color: "red"}})).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func httpPanel5() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(5).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("histogram_quantile(0.99, sum by (le) (rate(" + catalog.HTTPServerRequestDuration + "_bucket{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval])))").
			RefId("A")}).
		Title("P99 Latência").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 18, Y: 1}).
		Unit("s").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](0.5), Color: "yellow"},
				{Value: cog.ToPtr[float64](1), Color: "red"}})).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func httpRow6() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Rate").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 5}).
		Id(6)
}

func httpPanel7() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(7).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval]))").
			LegendFormat("{{job}}").
			RefId("A")}).
		Title("Requests por Serviço (req/s)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 6}).
		Unit("reqps").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](100), Color: "yellow"},
				{Value: cog.ToPtr[float64](500), Color: "red"}})).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none")).
		LineWidth(2).
		FillOpacity(10)
}

func httpPanel8() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(8).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (http_route, http_request_method) (rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval]))").
			LegendFormat("{{http_request_method}} {{http_route}}").
			RefId("A")}).
		Title("Requests por Rota (req/s)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 6}).
		Unit("reqps").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](100), Color: "yellow"},
				{Value: cog.ToPtr[float64](500), Color: "red"}})).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none")).
		LineWidth(2).
		FillOpacity(10)
}

func httpRow9() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Errors").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 14}).
		Id(9)
}

func httpPanel10() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(10).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("100 * (sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\", http_response_status_code=~\"5..\"}[$__rate_interval])) or sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval])) * 0) / sum by (job) (rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval]))").
			LegendFormat("{{job}}").
			RefId("A")}).
		Title("Error Rate por Serviço (%)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 15}).
		Unit("percent").
		Min(0).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](100), Color: "yellow"},
				{Value: cog.ToPtr[float64](500), Color: "red"}})).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none")).
		LineWidth(2).
		FillOpacity(10)
}

func httpPanel11() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(11).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (http_route, http_response_status_code) (rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\", http_response_status_code=~\"4..\"}[$__rate_interval]))").
			LegendFormat("{{http_response_status_code}} {{http_route}}").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("sum by (http_route, http_response_status_code) (rate(" + catalog.HTTPServerRequestTotal + "{job=~\"$service\", http_route!~\"/health|/ping\", http_response_status_code=~\"5..\"}[$__rate_interval]))").
				LegendFormat("{{http_response_status_code}} {{http_route}}").
				RefId("B")}).
		Title("Respostas 4xx/5xx por Rota (req/s)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 15}).
		Unit("reqps").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](100), Color: "yellow"},
				{Value: cog.ToPtr[float64](500), Color: "red"}})).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none")).
		LineWidth(2).
		FillOpacity(10)
}

func httpRow12() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Duration").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 23}).
		Id(12)
}

func httpPanel13() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(13).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("histogram_quantile(0.50, sum by (le) (rate(" + catalog.HTTPServerRequestDuration + "_bucket{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval])))").
			LegendFormat("P50").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("histogram_quantile(0.90, sum by (le) (rate(" + catalog.HTTPServerRequestDuration + "_bucket{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval])))").
				LegendFormat("P90").
				RefId("B"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("histogram_quantile(0.99, sum by (le) (rate(" + catalog.HTTPServerRequestDuration + "_bucket{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval])))").
				LegendFormat("P99").
				RefId("C")}).
		Title("Latência Global (P50 / P90 / P99)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 24}).
		Unit("s").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](100), Color: "yellow"},
				{Value: cog.ToPtr[float64](500), Color: "red"}})).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none")).
		LineWidth(2).
		FillOpacity(10)
}

func httpPanel14() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(14).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("histogram_quantile(0.99, sum by (le, http_route) (rate(" + catalog.HTTPServerRequestDuration + "_bucket{job=~\"$service\", http_route!~\"/health|/ping\"}[$__rate_interval])))").
			LegendFormat("{{http_route}}").
			RefId("A")}).
		Title("P99 por Rota").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 24}).
		Unit("s").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](100), Color: "yellow"},
				{Value: cog.ToPtr[float64](500), Color: "red"}})).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none")).
		LineWidth(2).
		FillOpacity(10)
}
