package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/nodegraph"
	"github.com/grafana/grafana-foundation-sdk/go/table"
	"github.com/grafana/grafana-foundation-sdk/go/tempo"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

const (
	bySLO = "max by (sloth_service, sloth_slo, service) "

	burnRateTicket = 1
	burnRatePage   = 6
	budgetLow      = 0.25
)

func overviewDashboard() *dashboard.DashboardBuilder {
	return newDashboard("overview").
		Tags([]string{config.Namespace, "overview"}).
		Readonly().
		Tooltip(1).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{datasourceVariable(tempoDS), datasourceVariable(prometheusDS)}).
		Annotations(grafanaAnnotations()).
		Links([]cog.Builder[dashboard.DashboardLink]{
			dashboard.NewDashboardLinkBuilder("Dashboards").
				Type("dashboards").
				Icon("external link").
				Tags([]string{config.Namespace}).
				KeepTime(true),
			dashboardLink("Kafka", "kafka"),
			dashboardLink("Postgres", "postgres"),
		}).
		WithPanel(sloTable("").Id(1).GridPos(dashboard.GridPos{H: 10, W: 24, X: 0, Y: 0})).
		WithPanel(alertList(2, "Alertas ativos",
			"Todos os alertas que estão disparando, pendentes ou com erro de avaliação, de todos os serviços e da infraestrutura.",
			everyAlert, false).
			GridPos(dashboard.GridPos{H: 12, W: 12, X: 0, Y: 10})).
		WithPanel(serviceMap())
}

func dashboardLink(title, name string) *dashboard.DashboardLinkBuilder {
	return dashboard.NewDashboardLinkBuilder(title).
		Type("link").
		Icon("dashboard").
		Url("/d/" + config.Namespace + "-" + name).
		KeepTime(true)
}

func sloTable(selector string) *table.PanelBuilder {
	percent := []dashboard.DynamicConfigValue{{Id: "unit", Value: "percentunit"}, {Id: "decimals", Value: 3}}
	of := func(series string) string { return bySLO + "(" + series + selector + ")" }

	return table.NewPanelBuilder().
		Title("SLOs").
		Description("Objetivo, SLI e budget restante de cada SLO na janela de 30 dias, e o burn rate atual "+
			"(1 = queima o budget exatamente em 30 dias). O nome do serviço abre o dashboard dele.").
		Datasource(datasourceRef(prometheusDS)).
		Targets(promTargets(
			promInstantTable("A", of(catalog.SLOObjectiveRatio)),
			promInstantTable("B", "1 - "+of(catalog.SLOErrorRatioRate30d)),
			promInstantTable("C", of(catalog.SLOPeriodErrorBudgetRemainingRatio)),
			promInstantTable("D", of(catalog.SLOCurrentBurnRateRatio)),
		)).
		Unit("percentunit").
		Transformations([]dashboard.DataTransformerConfig{
			{Id: "merge", Options: map[string]any{}},
			{Id: "organize", Options: map[string]any{
				"excludeByName": map[string]bool{"Time": true},
				"indexByName": map[string]int{
					"sloth_service": 0, "sloth_slo": 1, "service": 2,
					"Value #A": 3, "Value #B": 4, "Value #C": 5, "Value #D": 6,
				},
				"renameByName": map[string]string{
					"sloth_service": "Serviço", "sloth_slo": "SLO", "service": "Processo",
					"Value #A": "Objetivo", "Value #B": "SLI (30d)", "Value #C": "Budget restante (30d)",
					"Value #D": "Burn rate",
				},
			}},
		}).
		OverrideByName("Serviço", []dashboard.DynamicConfigValue{{Id: "links", Value: []map[string]any{{
			"title": "Dashboard do serviço",
			"url":   "/d/" + config.Namespace + "-${__value.raw}?${__url_time_range}",
		}}}}).
		OverrideByName("Objetivo", percent).
		OverrideByName("SLI (30d)", percent).
		OverrideByName("Budget restante (30d)", []dashboard.DynamicConfigValue{
			{Id: "custom.cellOptions", Value: map[string]any{"type": "color-background"}},
			{Id: "thresholds", Value: map[string]any{"mode": "absolute", "steps": []map[string]any{
				{"color": "red", "value": nil}, {"color": "yellow", "value": 0}, {"color": "green", "value": budgetLow},
			}}},
		}).
		OverrideByName("Burn rate", []dashboard.DynamicConfigValue{
			{Id: "unit", Value: "none"},
			{Id: "decimals", Value: 2},
			{Id: "custom.cellOptions", Value: map[string]any{"type": "color-text"}},
			{Id: "thresholds", Value: map[string]any{"mode": "absolute", "steps": []map[string]any{
				{"color": "green", "value": nil}, {"color": "yellow", "value": burnRateTicket}, {"color": "red", "value": burnRatePage},
			}}},
		})
}

func serviceMap() cog.Builder[dashboard.Panel] {
	return nodegraph.NewPanelBuilder().
		Id(3).
		Title("Service map").
		Description("Chamadas entre serviços vistas nos traces, com taxa e latência, a partir do metrics-generator do Tempo.").
		Datasource(datasourceRef(tempoDS)).
		Targets([]cog.Builder[variants.Dataquery]{tempo.NewDataqueryBuilder().
			RefId("A").
			QueryType("serviceMap").
			Datasource(datasourceRef(tempoDS))}).
		GridPos(dashboard.GridPos{H: 12, W: 12, X: 12, Y: 10})
}
