package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

func grafanaAnnotations() []cog.Builder[dashboard.AnnotationQuery] {
	return []cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
		Name("Annotations & Alerts").
		Datasource(common.DataSourceRef{Type: cog.ToPtr("grafana"), Uid: cog.ToPtr("-- Grafana --")}).
		Hide(true).
		IconColor("rgba(0, 211, 255, 1)").
		Type("dashboard").
		BuiltIn(1)}
}

func promQuery(refID, expr, legend string) *prometheus.DataqueryBuilder {
	return prometheus.NewDataqueryBuilder().
		Datasource(datasourceRef(prometheusDS)).
		Expr(expr).
		LegendFormat(legend).
		RefId(refID)
}

func promInstantTable(refID, expr string) *prometheus.DataqueryBuilder {
	return prometheus.NewDataqueryBuilder().
		Datasource(datasourceRef(prometheusDS)).
		Expr(expr).
		Instant().
		Format(prometheus.PromQueryFormatTable).
		RefId(refID)
}

func promTargets(builders ...*prometheus.DataqueryBuilder) []cog.Builder[variants.Dataquery] {
	out := make([]cog.Builder[variants.Dataquery], 0, len(builders))
	for _, b := range builders {
		out = append(out, b)
	}

	return out
}

func row(id uint32, title string, y uint32) *dashboard.RowBuilder {
	return dashboard.NewRowBuilder(title).
		Id(id).
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: y})
}

func timeseriesPanel(id uint32, title, description, unit string, pos dashboard.GridPos,
	targets ...*prometheus.DataqueryBuilder,
) *timeseries.PanelBuilder {
	return timeseries.NewPanelBuilder().
		Id(id).
		Title(title).
		Description(description).
		Datasource(datasourceRef(prometheusDS)).
		Targets(promTargets(targets...)).
		GridPos(pos).
		Unit(unit).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("multi").
			Sort("desc")).
		DrawStyle("line").
		LineWidth(1).
		FillOpacity(10).
		ShowPoints("never")
}

func statPanel(id uint32, title, description, unit string, pos dashboard.GridPos, expr string,
	steps []dashboard.Threshold,
) *stat.PanelBuilder {
	return stat.NewPanelBuilder().
		Id(id).
		Title(title).
		Description(description).
		Datasource(datasourceRef(prometheusDS)).
		Targets(promTargets(promQuery("A", expr, ""))).
		GridPos(pos).
		Unit(unit).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps(steps)).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("thresholds")).
		ColorMode("background").
		GraphMode("none").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"}))
}
