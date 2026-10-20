package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/logs"
	"github.com/grafana/grafana-foundation-sdk/go/loki"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

const (
	fullWidth  = 24
	halfWidth  = 12
	thirdWidth = 8
	quarter    = 6

	statHeight  = 5
	graphHeight = 8
	listHeight  = 10
)

type panel interface {
	build(id uint32) cog.Builder[dashboard.Panel]
}

type query struct {
	expr     string
	legend   string
	exemplar bool
}

func (q query) prometheus(refID string) *prometheus.DataqueryBuilder {
	target := prometheus.NewDataqueryBuilder().
		Datasource(datasourceRef(prometheusDS)).
		Expr(q.expr).
		LegendFormat(q.legend).
		RefId(refID)
	if q.exemplar {
		target.Exemplar(true)
	}

	return target
}

const refIDs = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func refID(i int) string {
	return refIDs[i : i+1]
}

func prometheusTargets(queries []query) []cog.Builder[variants.Dataquery] {
	out := make([]cog.Builder[variants.Dataquery], 0, len(queries))
	for i, q := range queries {
		out = append(out, q.prometheus(refID(i)))
	}

	return out
}

type step struct {
	from  *float64
	color string
}

func at(value float64, color string) step {
	return step{from: &value, color: color}
}

func thresholds(steps []step) *dashboard.ThresholdsConfigBuilder {
	out := make([]dashboard.Threshold, 0, len(steps))
	for _, s := range steps {
		out = append(out, dashboard.Threshold{Value: s.from, Color: s.color})
	}

	return dashboard.NewThresholdsConfigBuilder().Mode(dashboard.ThresholdsModeAbsolute).Steps(out)
}

type graph struct {
	title       string
	description string
	unit        string
	width       uint32
	queries     []query
	sloLine     *float64
	links       []cog.Builder[dashboard.DashboardLink]
	logQL       bool
}

func (g *graph) build(id uint32) cog.Builder[dashboard.Panel] {
	ds := prometheusDS
	targets := prometheusTargets(g.queries)

	if g.logQL {
		ds = lokiDS
		targets = targets[:0]

		for i, q := range g.queries {
			targets = append(targets, loki.NewDataqueryBuilder().
				Datasource(datasourceRef(lokiDS)).
				Expr(q.expr).
				LegendFormat(q.legend).
				RefId(refID(i)))
		}
	}

	out := timeseries.NewPanelBuilder().
		Id(id).
		Title(g.title).
		Description(g.description).
		Datasource(datasourceRef(ds)).
		Targets(targets).
		GridPos(dashboard.GridPos{W: g.width, H: graphHeight}).
		Unit(g.unit).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode(common.LegendDisplayModeList).
			Placement(common.LegendPlacementBottom).
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode(common.TooltipDisplayModeMulti).
			Sort(common.SortOrderDescending)).
		DrawStyle(common.GraphDrawStyleLine).
		LineWidth(1).
		FillOpacity(10).
		ShowPoints(common.VisibilityModeNever).
		SpanNulls(common.BoolOrFloat64{Bool: cog.ToPtr(false)})

	if g.sloLine != nil {
		out.Thresholds(thresholds([]step{{color: "green"}, at(*g.sloLine, "red")})).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().Mode(common.GraphThresholdsStyleModeLine)).
			AxisSoftMax(*g.sloLine)
	}

	if len(g.links) > 0 {
		out.DataLinks(g.links)
	}

	return out
}

type single struct {
	title       string
	description string
	unit        string
	width       uint32
	queries     []query
	steps       []step
}

func (s *single) build(id uint32) cog.Builder[dashboard.Panel] {
	out := stat.NewPanelBuilder().
		Id(id).
		Title(s.title).
		Description(s.description).
		Datasource(datasourceRef(prometheusDS)).
		Targets(prometheusTargets(s.queries)).
		GridPos(dashboard.GridPos{W: s.width, H: statHeight}).
		Unit(s.unit).
		GraphMode(common.BigValueGraphModeArea).
		ColorMode(common.BigValueColorModeValue).
		JustifyMode(common.BigValueJustifyModeAuto).
		TextMode(common.BigValueTextModeAuto).
		ReduceOptions(common.NewReduceDataOptionsBuilder().Calcs([]string{"lastNotNull"})).
		Orientation(common.VizOrientationAuto)

	if len(s.steps) > 0 {
		out.Thresholds(thresholds(s.steps)).
			ColorScheme(dashboard.NewFieldColorBuilder().Mode(dashboard.FieldColorModeIdThresholds))
	}

	return out
}

type serviceAlerts struct {
	title       string
	description string
	filter      string
}

func (a serviceAlerts) build(id uint32) cog.Builder[dashboard.Panel] {
	return alertList(id, a.title, a.description, a.filter, true).GridPos(dashboard.GridPos{W: fullWidth, H: listHeight})
}

type logLines struct {
	title       string
	description string
	expr        string
}

func (l logLines) build(id uint32) cog.Builder[dashboard.Panel] {
	return logs.NewPanelBuilder().
		Id(id).
		Title(l.title).
		Description(l.description).
		Datasource(datasourceRef(lokiDS)).
		Targets([]cog.Builder[variants.Dataquery]{loki.NewDataqueryBuilder().
			Datasource(datasourceRef(lokiDS)).
			Expr(l.expr).
			RefId("A")}).
		GridPos(dashboard.GridPos{W: fullWidth, H: listHeight}).
		ShowTime(true).
		ShowLabels(true).
		WrapLogMessage(true).
		EnableLogDetails(true).
		SortOrder(common.LogsSortOrderDescending).
		DedupStrategy(common.LogsDedupStrategyNone)
}
