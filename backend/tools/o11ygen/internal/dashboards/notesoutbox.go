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

func notesOutboxDashboard() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder("Notes — Outbox · Forward").
		Uid(config.Namespace+"-notes-outbox").
		Tags([]string{"messaging",
			"notes",
			config.Namespace,
			"sre"}).
		Editable().
		Tooltip(1).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{datasourceVariable(prometheusDS),
			dashboard.NewQueryVariableBuilder("topic").
				Label("Tópico").
				Query(dashboard.StringOrMap{Map: map[string]any{"query": "label_values(" + catalog.KafkaTopicPartitionCurrentOffset + ", topic)", "refId": "topic-var"}}).
				Datasource(datasourceRef(prometheusDS)).
				Current(dashboard.VariableOption{Selected: cog.ToPtr[bool](true), Text: dashboard.StringOrArrayOfString{String: cog.ToPtr[string]("All")}, Value: dashboard.StringOrArrayOfString{String: cog.ToPtr[string]("$__all")}}).
				Multi(true).
				Refresh(2).
				Sort(1).
				IncludeAll(true).
				AllValue(".+").
				Regex("/^(?!.*\\.dlq$).*/").
				Definition("label_values(" + catalog.KafkaTopicPartitionCurrentOffset + ", topic)")}).
		Annotations([]cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
			Name("Annotations & Alerts").
			Datasource(common.DataSourceRef{Type: cog.ToPtr[string]("grafana"), Uid: cog.ToPtr[string]("-- Grafana --")}).
			Hide(true).
			IconColor("rgba(0, 211, 255, 1)").
			Type("dashboard").
			BuiltIn(1)}).
		WithRow(notesOutboxRow3()).
		WithPanel(notesOutboxPanel4()).
		WithPanel(notesOutboxPanel5()).
		WithPanel(notesOutboxPanel6()).
		WithPanel(notesOutboxPanel7()).
		WithPanel(notesOutboxPanel8()).
		WithPanel(notesOutboxPanel9()).
		WithRow(notesOutboxRow10()).
		WithPanel(notesOutboxPanel11()).
		WithPanel(notesOutboxPanel12())
}

func notesOutboxRow3() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Outbox").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 0}).
		Id(3)
}

func notesOutboxPanel4() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(4).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("max(" + catalog.NotesOutboxOldestPendingAge + ")").
			RefId("A")}).
		Title("Pendente mais antiga").
		Description("Idade da mensagem mais antiga do outbox ainda não levada ao Kafka; 0 sem pendentes. O alerta dispara acima de 120s por 5 min.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 0, Y: 1}).
		Unit("s").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](60), Color: "yellow"},
				{Value: cog.ToPtr[float64](120), Color: "red"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("thresholds")).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func notesOutboxPanel5() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(5).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum(increase(" + catalog.NotesOutboxPurgedMessages + "[24h])) or vector(0)").
			RefId("A")}).
		Title("Encaminhadas apagadas (24h)").
		Description("Mensagens já encaminhadas há mais de 7 dias que a limpeza apagou.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 6, Y: 1}).
		Unit("short").
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func notesOutboxPanel6() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(6).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum(increase(" + catalog.NotesOutboxPurgeRuns + "{outcome=\"failure\"}[24h])) or vector(0)").
			RefId("A")}).
		Title("Limpezas com falha (24h)").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 12, Y: 1}).
		Unit("short").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "green"},
				{Value: cog.ToPtr[float64](1), Color: "red"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("thresholds")).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func notesOutboxPanel7() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(7).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("count(" + catalog.NotesOutboxOldestPendingAge + ") or vector(0)").
			RefId("A")}).
		Title("Réplicas do forwarder reportando").
		Description("Réplicas que reportam o gauge; 0 significa que nenhum forwarder está esvaziando o outbox.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 18, Y: 1}).
		Unit("short").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Color: "red"},
				{Value: cog.ToPtr[float64](1), Color: "green"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("thresholds")).
		GraphMode("area").
		ColorMode("background").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Calcs([]string{"lastNotNull"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func notesOutboxPanel8() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(8).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("max by (job) (" + catalog.NotesOutboxOldestPendingAge + ")").
			LegendFormat("{{job}}").
			RefId("A")}).
		Title("Idade da pendente mais antiga").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 5}).
		Unit("s").
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
		ShowPoints("never").
		SpanNulls(common.BoolOrFloat64{Bool: cog.ToPtr[bool](false)})
}

func notesOutboxPanel9() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(9).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (outcome) (increase(" + catalog.NotesOutboxPurgeRuns + "[$__rate_interval]))").
			LegendFormat("execuções {{outcome}}").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("sum(increase(" + catalog.NotesOutboxPurgedMessages + "[$__rate_interval]))").
				LegendFormat("mensagens apagadas").
				RefId("B")}).
		Title("Limpeza").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 5}).
		Unit("short").
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
		ShowPoints("never").
		SpanNulls(common.BoolOrFloat64{Bool: cog.ToPtr[bool](false)})
}

func notesOutboxRow10() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Forward").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 13}).
		Id(10)
}

func notesOutboxPanel11() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(11).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (messaging_destination_name) (rate(" + catalog.MessagingClientSentMessages + "{messaging_destination_name=~\"$topic\", error_type=\"\"}[$__rate_interval]))").
			LegendFormat("ok {{messaging_destination_name}}").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("sum by (messaging_destination_name, error_type) (rate(" + catalog.MessagingClientSentMessages + "{messaging_destination_name=~\"$topic\", error_type!=\"\"}[$__rate_interval]))").
				LegendFormat("{{error_type}} {{messaging_destination_name}}").
				RefId("B")}).
		Title("Envios ao Kafka (msg/s)").
		Description("Cada tentativa do forwarder; erros entram em retry infinito, sem pular mensagens.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 14}).
		Unit("ops").
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
		ShowPoints("never").
		SpanNulls(common.BoolOrFloat64{Bool: cog.ToPtr[bool](false)})
}

func notesOutboxPanel12() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(12).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("histogram_quantile(0.5, sum by (le) (rate(" + catalog.MessagingClientOperationDuration + "_bucket{messaging_destination_name=~\"$topic\"}[$__rate_interval])))").
			LegendFormat("P50").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("histogram_quantile(0.99, sum by (le) (rate(" + catalog.MessagingClientOperationDuration + "_bucket{messaging_destination_name=~\"$topic\"}[$__rate_interval])))").
				LegendFormat("P99").
				RefId("B")}).
		Title("Duração do envio").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 14}).
		Unit("s").
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
		ShowPoints("never").
		SpanNulls(common.BoolOrFloat64{Bool: cog.ToPtr[bool](false)})
}
