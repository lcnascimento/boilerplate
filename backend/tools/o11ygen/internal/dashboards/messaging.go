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

func messagingDashboard() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder("Mensageria — Produce · Consume · DLQ").
		Uid(config.Namespace+"-messaging").
		Tags([]string{"messaging",
			"kafka",
			config.Namespace,
			"sre"}).
		Readonly().
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
				Definition("label_values(" + catalog.KafkaTopicPartitionCurrentOffset + ", topic)"),
			dashboard.NewQueryVariableBuilder("group").
				Label("Consumer group").
				Query(dashboard.StringOrMap{Map: map[string]any{"query": "label_values(" + catalog.KafkaConsumerGroupUncommittedOffsets + ", consumergroup)", "refId": "group-var"}}).
				Datasource(datasourceRef(prometheusDS)).
				Current(dashboard.VariableOption{Selected: cog.ToPtr[bool](true), Text: dashboard.StringOrArrayOfString{String: cog.ToPtr[string]("All")}, Value: dashboard.StringOrArrayOfString{String: cog.ToPtr[string]("$__all")}}).
				Multi(true).
				Refresh(2).
				Sort(1).
				IncludeAll(true).
				AllValue(".+").
				Definition("label_values(" + catalog.KafkaConsumerGroupUncommittedOffsets + ", consumergroup)")}).
		Annotations([]cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
			Name("Annotations & Alerts").
			Datasource(common.DataSourceRef{Type: cog.ToPtr[string]("grafana"), Uid: cog.ToPtr[string]("-- Grafana --")}).
			Hide(true).
			IconColor("rgba(0, 211, 255, 1)").
			Type("dashboard").
			BuiltIn(1)}).
		WithRow(messagingRow1()).
		WithPanel(messagingPanel2()).
		WithRow(messagingRow10()).
		WithPanel(messagingPanel13()).
		WithRow(messagingRow14()).
		WithPanel(messagingPanel15()).
		WithPanel(messagingPanel16()).
		WithPanel(messagingPanel17()).
		WithPanel(messagingPanel18()).
		WithPanel(messagingPanel19()).
		WithRow(messagingRow20()).
		WithPanel(messagingPanel21()).
		WithPanel(messagingPanel22()).
		WithPanel(messagingPanel23()).
		WithPanel(messagingPanel24())
}

func messagingRow1() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Alertas").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 0}).
		Id(1)
}

func messagingPanel2() cog.Builder[dashboard.Panel] {
	return dashboard.NewPanelBuilder().
		Type("alertlist").
		Id(2).
		Title("Estado dos alertas de mensageria").
		GridPos(dashboard.GridPos{H: 9, W: 24, X: 0, Y: 1}).
		Options(map[string]any{
			"viewMode":                 "list",
			"groupMode":                "default",
			"groupBy":                  []string{},
			"maxItems":                 20,
			"sortOrder":                1,
			"dashboardAlerts":          false,
			"alertName":                "",
			"alertInstanceLabelFilter": `{area="messaging"}`,
			"stateFilter": map[string]bool{
				"firing":     true,
				"pending":    true,
				"noData":     true,
				"normal":     true,
				"error":      true,
				"recovering": true,
			},
			"showInstances": false,
		})
}

func messagingRow10() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Produce").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 10}).
		Id(10)
}

func messagingPanel13() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(13).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (topic) (rate(" + catalog.KafkaTopicPartitionCurrentOffset + "{topic=~\"$topic\"}[$__rate_interval]))").
			LegendFormat("{{topic}}").
			RefId("A")}).
		Title("Produção no tópico (msg/s)").
		Description("Avanço dos high-water marks no broker, via o exporter Kafka do Alloy.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 24, X: 0, Y: 11}).
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

func messagingRow14() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("Consume").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 19}).
		Id(14)
}

func messagingPanel15() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(15).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (consumergroup, topic) (" + catalog.KafkaConsumerGroupUncommittedOffsets + "{consumergroup=~\"$group\", topic=~\"$topic\"})").
			LegendFormat("{{consumergroup}} {{topic}}").
			RefId("A")}).
		Title("Lag por consumer group").
		Description("Mensagens ainda não confirmadas pelo group. O alerta do group audit dispara acima de 100 por 5 min.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 8, X: 0, Y: 20}).
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

func messagingPanel16() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(16).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (messaging_consumer_group_name) (rate(" + catalog.MessagingClientConsumedMessages + "{messaging_consumer_group_name=~\"$group\", messaging_destination_name=~\"$topic\", error_type=\"\"}[$__rate_interval]))").
			LegendFormat("ok {{messaging_consumer_group_name}}").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("sum by (messaging_consumer_group_name, error_type) (rate(" + catalog.MessagingClientConsumedMessages + "{messaging_consumer_group_name=~\"$group\", messaging_destination_name=~\"$topic\", error_type!=\"\"}[$__rate_interval]))").
				LegendFormat("{{error_type}} {{messaging_consumer_group_name}}").
				RefId("B")}).
		Title("Mensagens consumidas (msg/s)").
		Description("Mensagens que os consumers terminaram de tratar por segundo: sucesso e falha por error_type.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 8, X: 8, Y: 20}).
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

func messagingPanel17() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(17).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("histogram_quantile(0.5, sum by (le, messaging_consumer_group_name) (rate(" + catalog.MessagingProcessDuration + "_bucket{messaging_consumer_group_name=~\"$group\", messaging_destination_name=~\"$topic\"}[$__rate_interval])))").
			LegendFormat("P50 {{messaging_consumer_group_name}}").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("histogram_quantile(0.99, sum by (le, messaging_consumer_group_name) (rate(" + catalog.MessagingProcessDuration + "_bucket{messaging_consumer_group_name=~\"$group\", messaging_destination_name=~\"$topic\"}[$__rate_interval])))").
				LegendFormat("P99 {{messaging_consumer_group_name}}").
				RefId("B")}).
		Title("Duração do processamento").
		Description("Percentis 50 e 99 da duração de uma tentativa de processamento, por consumer group.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 8, X: 16, Y: 20}).
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

func messagingPanel18() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(18).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (messaging_consumer_group_name, error_type) (rate(" + catalog.MessagingProcessDuration + "_count{messaging_consumer_group_name=~\"$group\", messaging_destination_name=~\"$topic\", error_type!=\"\"}[$__rate_interval]))").
			LegendFormat("{{error_type}} {{messaging_consumer_group_name}}").
			RefId("A")}).
		Title("Retentativas (tentativas/s)").
		Description("Tentativas de processamento com erro; erros transitórios bloqueiam a partição até passar.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 28}).
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

func messagingPanel19() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(19).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (consumergroup, topic) (" + catalog.KafkaConsumerGroupCurrentOffset + "{consumergroup=~\"$group\", topic=~\"$topic\"})").
			LegendFormat("confirmado {{consumergroup}}").
			RefId("A"),
			prometheus.NewDataqueryBuilder().
				Datasource(datasourceRef(prometheusDS)).
				Expr("sum by (topic) (" + catalog.KafkaTopicPartitionCurrentOffset + "{topic=~\"$topic\"})").
				LegendFormat("high-water {{topic}}").
				RefId("B")}).
		Title("Offset confirmado vs. high-water mark").
		Description("Offset confirmado de cada consumer group comparado ao último offset de cada tópico.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 28}).
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

func messagingRow20() *dashboard.RowBuilder {
	return dashboard.NewRowBuilder("DLQ").
		GridPos(dashboard.GridPos{H: 1, W: 24, X: 0, Y: 36}).
		Id(20)
}

func messagingPanel21() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(21).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum(" + catalog.KafkaTopicPartitionCurrentOffset + "{topic=~\".+\\\\.dlq\"}) or vector(0)").
			RefId("A")}).
		Title("Mensagens nas DLQs").
		Description("High-water mark das DLQs (retenção infinita): tudo que já caiu nelas.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 0, Y: 37}).
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

func messagingPanel22() cog.Builder[dashboard.Panel] {
	return stat.NewPanelBuilder().
		Id(22).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum(increase(" + catalog.KafkaTopicPartitionCurrentOffset + "{topic=~\".+\\\\.dlq\"}[5m])) or vector(0)").
			RefId("A")}).
		Title("Novas na DLQ (5 min)").
		Description("O alerta de DLQ dispara assim que este valor passa de 0.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 4, W: 6, X: 6, Y: 37}).
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

func messagingPanel23() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(23).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (messaging_consumer_group_name, error_type) (increase(" + catalog.MessagingDLQSentMessages + "{messaging_consumer_group_name=~\"$group\", messaging_destination_name=~\"$topic\"}[$__rate_interval]))").
			LegendFormat("{{error_type}} {{messaging_consumer_group_name}}").
			RefId("A")}).
		Title("Enviadas à DLQ por causa").
		Description("Mensagens enviadas à DLQ por consumer group e error_type.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 0, Y: 41}).
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
		Stacking(common.NewStackingConfigBuilder().
			Mode("normal").
			Group("A")).
		SpanNulls(common.BoolOrFloat64{Bool: cog.ToPtr[bool](false)})
}

func messagingPanel24() cog.Builder[dashboard.Panel] {
	return timeseries.NewPanelBuilder().
		Id(24).
		Targets([]cog.Builder[variants.Dataquery]{prometheus.NewDataqueryBuilder().
			Datasource(datasourceRef(prometheusDS)).
			Expr("sum by (topic) (" + catalog.KafkaTopicPartitionCurrentOffset + "{topic=~\".+\\\\.dlq\"})").
			LegendFormat("{{topic}}").
			RefId("A")}).
		Title("High-water mark das DLQs").
		Description("Último offset de cada tópico de DLQ; um degrau é uma mensagem nova na DLQ.").
		Datasource(datasourceRef(prometheusDS)).
		GridPos(dashboard.GridPos{H: 8, W: 12, X: 12, Y: 41}).
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
