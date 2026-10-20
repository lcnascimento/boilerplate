package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

type boardTarget struct {
	Expr     string `json:"expr"`
	Exemplar bool   `json:"exemplar"`
}

type boardLink struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	KeepTime bool   `json:"keepTime"` //nolint:tagliatelle // Grafana's dashboard schema.
}

type boardPanel struct {
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Collapsed   bool           `json:"collapsed"`
	Targets     []boardTarget  `json:"targets"`
	Panels      []boardPanel   `json:"panels"`
	Options     map[string]any `json:"options"`
	FieldConfig struct {
		Defaults struct {
			Links      []boardLink `json:"links"`
			Thresholds struct {
				Steps []struct {
					Value *float64 `json:"value"`
				} `json:"steps"`
			} `json:"thresholds"`
			Custom struct {
				ThresholdsStyle struct {
					Mode string `json:"mode"`
				} `json:"thresholdsStyle"` //nolint:tagliatelle // Grafana's dashboard schema.
			} `json:"custom"`
		} `json:"defaults"`
	} `json:"fieldConfig"` //nolint:tagliatelle // Grafana's dashboard schema.
}

type board struct {
	UID        string `json:"uid"`
	Templating struct {
		List []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Query   any    `json:"query"`
			Multi   bool   `json:"multi"`
			Include bool   `json:"includeAll"` //nolint:tagliatelle // Grafana's dashboard schema.
		} `json:"list"`
	} `json:"templating"`
	Links  []boardLink  `json:"links"`
	Panels []boardPanel `json:"panels"`
}

type section struct {
	row    boardPanel
	panels []boardPanel
}

func readBoard(t *testing.T, root, name string) board {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, dashboardsDir, name)) //nolint:gosec // Test temporary directory.
	require.NoError(t, err)

	var out board
	require.NoError(t, json.Unmarshal(raw, &out))

	return out
}

func (b *board) sections() []section {
	var out []section

	for i := range b.Panels {
		panel := &b.Panels[i]
		if panel.Type == "row" {
			out = append(out, section{row: *panel, panels: panel.Panels})

			continue
		}

		out[len(out)-1].panels = append(out[len(out)-1].panels, *panel)
	}

	return out
}

func (b *board) section(t *testing.T, title string) section {
	t.Helper()

	sections := b.sections()
	for i := range sections {
		if sections[i].row.Title == title {
			return sections[i]
		}
	}

	require.Failf(t, "missing section", "no row %q", title)

	return section{}
}

func (b *board) everyPanel() []boardPanel {
	var out []boardPanel

	sections := b.sections()
	for i := range sections {
		out = append(out, sections[i].panels...)
	}

	return out
}

func generateOrders(t *testing.T) (string, board) {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{ordersService}))

	return root, readBoard(t, root, "orders.json")
}

func TestGenerate_WritesOneDashboardPerServiceUnderTheUIDItsAlertsPointTo(t *testing.T) {
	root, orders := generateOrders(t)

	assert.Equal(t, "boilerplate-orders", orders.UID)

	pointed := map[string]string{}

	for _, file := range []string{filepath.Join(sloRulesDir, "orders.yaml"), filepath.Join(rulesDir, "orders.yaml")} {
		for _, group := range readRules(t, filepath.Join(root, file)).Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					pointed[rule.Alert] = rule.Annotations["dashboard_url"]
				}
			}
		}
	}

	ordersBoard := "http://localhost:3010/d/boilerplate-orders"

	assert.Equal(t, map[string]string{
		"SLOBurnRate":            ordersBoard,
		"ServiceTelemetryAbsent": ordersBoard,
		"CriticalLogRecorded":    ordersBoard,
		"DBPoolSaturated":        ordersBoard,
		"ConsumerLagHigh":        ordersBoard,
		"DLQMessageReceived":     ordersBoard,
		"OutboxAgeAbsent":        ordersBoard,
		"SeriesBudgetExceeded":   "http://localhost:3010/d/boilerplate-postgres",
	}, pointed)
}

func TestGenerate_ServiceDashboardStacksSLOsComponentsAndCommonSections(t *testing.T) {
	_, orders := generateOrders(t)

	sections := orders.sections()
	rows := make([]string, 0, len(sections))

	for i := range sections {
		state := "open"
		if sections[i].row.Collapsed {
			state = "collapsed"
		}

		rows = append(rows, sections[i].row.Title+" "+state)
	}

	assert.Equal(t, []string{
		"SLOs e alertas open", "api open", "consumer open", "worker open", "Runtime collapsed", "Banco collapsed",
		"Logs open",
	}, rows)
}

func TestGenerate_LeavesOutTheSectionsAServiceDoesNotRun(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{{
		Name:      "billing",
		Processes: []service.Process{service.Daemon("billing-consumer", service.Consumer).WithConsumerGroup("billing")},
	}}))

	billing := readBoard(t, root, "billing.json")

	sections := billing.sections()
	rows := make([]string, 0, len(sections))

	for i := range sections {
		rows = append(rows, sections[i].row.Title)
	}

	assert.Equal(t, []string{"SLOs e alertas", "consumer", "Runtime", "Logs"}, rows)
}

func TestGenerate_ServiceDashboardFiltersProcessesWithoutAServiceVariable(t *testing.T) {
	_, orders := generateOrders(t)

	variables := make([]string, 0, len(orders.Templating.List))

	for _, v := range orders.Templating.List {
		variables = append(variables, v.Type+" "+v.Name)

		if v.Name == "process" {
			assert.Equal(t, "orders-api,orders-consumer,orders-worker", v.Query)
			assert.True(t, v.Multi)
			assert.True(t, v.Include)
		}
	}

	assert.Equal(t, []string{
		"datasource tempo_datasource", "datasource prometheus_datasource", "datasource loki_datasource",
		"custom process",
	}, variables)
}

func TestGenerate_ServiceDashboardTopsWithTheSlothSLOsAndTheServiceAlerts(t *testing.T) {
	_, orders := generateOrders(t)

	top := orders.section(t, "SLOs e alertas")

	queries := map[string]string{}
	filter := ""

	for _, panel := range top.panels {
		for _, target := range panel.Targets {
			queries[panel.Title] = target.Expr
		}

		if panel.Type == "alertlist" {
			filter, _ = panel.Options["alertInstanceLabelFilter"].(string)
		}
	}

	assert.Equal(t, map[string]string{
		"SLI atual":             `1 - max by (sloth_slo) (slo:sli_error:ratio_rate5m{sloth_service="orders"})`,
		"Budget restante (30d)": `max by (sloth_slo) (slo:period_error_budget_remaining:ratio{sloth_service="orders"})`,
		"Burn rate":             `max by (sloth_slo) (slo:current_burn_rate:ratio{sloth_service="orders"})`,
	}, queries)
	assert.Equal(t, `{service=~"orders-.*"}`, filter)
}

func TestGenerate_ServiceDashboardAsksForExemplarsOnlyOnLatencyHistograms(t *testing.T) {
	_, orders := generateOrders(t)

	var exemplars []string

	for _, panel := range orders.everyPanel() {
		for _, target := range panel.Targets {
			if target.Exemplar {
				exemplars = append(exemplars, panel.Title+": "+target.Expr)
			}
		}
	}

	assert.Equal(t, []string{
		`Latência p50/p99: histogram_quantile(0.5, sum by (le) (rate(http_server_request_duration_seconds_bucket` +
			`{job=~"boilerplate/(orders-api)", http_route!~"/ping|/health"}[$__rate_interval])))`,
		`Latência p50/p99: histogram_quantile(0.99, sum by (le) (rate(http_server_request_duration_seconds_bucket` +
			`{job=~"boilerplate/(orders-api)", http_route!~"/ping|/health"}[$__rate_interval])))`,
		`Latência p99 por rota: histogram_quantile(0.99, sum by (le, http_route) (rate(` +
			`http_server_request_duration_seconds_bucket{job=~"boilerplate/(orders-api)", http_route!~"/ping|/health"}` +
			`[$__rate_interval])))`,
		`Duração do processamento p50/p99: histogram_quantile(0.5, sum by (le) (rate(` +
			`messaging_process_duration_seconds_bucket{job=~"boilerplate/(orders-consumer)"}[$__rate_interval])))`,
		`Duração do processamento p50/p99: histogram_quantile(0.99, sum by (le) (rate(` +
			`messaging_process_duration_seconds_bucket{job=~"boilerplate/(orders-consumer)"}[$__rate_interval])))`,
		`Duração do envio p50/p99: histogram_quantile(0.5, sum by (le) (rate(` +
			`messaging_client_operation_duration_seconds_bucket{job=~"boilerplate/(orders-worker)"}[$__rate_interval])))`,
		`Duração do envio p50/p99: histogram_quantile(0.99, sum by (le) (rate(` +
			`messaging_client_operation_duration_seconds_bucket{job=~"boilerplate/(orders-worker)"}[$__rate_interval])))`,
	}, exemplars)
}

func TestGenerate_ServiceDashboardDrawsTheSLOThresholdOnLatencyAndFreshness(t *testing.T) {
	_, orders := generateOrders(t)

	lines := map[string]float64{}

	for _, panel := range orders.everyPanel() {
		if panel.FieldConfig.Defaults.Custom.ThresholdsStyle.Mode != "line" {
			continue
		}

		steps := panel.FieldConfig.Defaults.Thresholds.Steps
		require.Len(t, steps, 2, panel.Title)
		require.NotNil(t, steps[1].Value, panel.Title)
		lines[panel.Title] = *steps[1].Value
	}

	assert.Equal(t, map[string]float64{
		"Latência p50/p99":                 0.25,
		"Latência p99 por rota":            0.25,
		"Duração do processamento p50/p99": 1,
		"Idade da pendente mais antiga":    30,
	}, lines)
}

func TestGenerate_ServiceDashboardLinksToTheDrilldownAppsOfItsProcesses(t *testing.T) {
	_, orders := generateOrders(t)

	links := map[string]string{}

	for _, link := range orders.Links {
		assert.True(t, link.KeepTime, link.Title)
		links[link.Title] = link.URL
	}

	logs := func(process string) string {
		return "/a/grafana-lokiexplore-app/explore/service_name/" + process + "/logs?var-ds=${loki_datasource}" +
			"&var-filters=service_name%7C%3D%7C" + process
	}
	profiles := func(process string) string {
		return "/a/grafana-pyroscope-app/explore?explorationType=flame-graph&var-serviceName=" + process +
			"&var-profileMetricId=process_cpu%3Acpu%3Ananoseconds%3Acpu%3Ananoseconds"
	}

	assert.Equal(t, map[string]string{
		"Traces": "/a/grafana-exploretraces-app/explore?var-ds=${tempo_datasource}" +
			"&var-filters=resource.service.name%7C%3D~%7Corders-api__gfp__orders-consumer__gfp__orders-worker" +
			"&var-groupBy=resource.service.name",
		"Logs orders-api":          logs("orders-api"),
		"Logs orders-consumer":     logs("orders-consumer"),
		"Logs orders-worker":       logs("orders-worker"),
		"Profiles orders-api":      profiles("orders-api"),
		"Profiles orders-consumer": profiles("orders-consumer"),
		"Profiles orders-worker":   profiles("orders-worker"),
	}, links)
}

func TestGenerate_ServiceDashboardLinksARouteToItsSlowTraces(t *testing.T) {
	_, orders := generateOrders(t)

	var links []boardLink

	for _, panel := range orders.section(t, "api").panels {
		links = append(links, panel.FieldConfig.Defaults.Links...)
	}

	require.Len(t, links, 1)
	assert.Equal(t, "Traces lentos desta rota", links[0].Title)

	explore, err := url.Parse(links[0].URL)
	require.NoError(t, err)
	assert.Equal(t, "/explore", explore.Path)

	var panes map[string]struct {
		Datasource string `json:"datasource"`
		Queries    []struct {
			Query     string `json:"query"`
			QueryType string `json:"queryType"` //nolint:tagliatelle // Grafana's dashboard schema.
		} `json:"queries"`
		Range map[string]string `json:"range"`
	}
	require.NoError(t, json.Unmarshal([]byte(explore.Query().Get("panes")), &panes))
	require.Len(t, panes["slow"].Queries, 1)

	assert.Equal(t, "${tempo_datasource}", panes["slow"].Datasource)
	assert.Equal(t, "traceql", panes["slow"].Queries[0].QueryType)
	assert.Equal(t, `{resource.service.name=~"orders-api" && kind=server && span.http.route="${__field.labels.http_route}" `+
		`&& duration > 250ms}`, panes["slow"].Queries[0].Query)
	assert.Equal(t, map[string]string{"from": "${__from}", "to": "${__to}"}, panes["slow"].Range)
}

func TestGenerate_RejectsAServiceNamedAfterAnInfrastructureDashboard(t *testing.T) {
	err := generate(t.TempDir(), []service.Service{{
		Name:      "overview",
		Processes: []service.Process{service.Daemon("overview-api", service.API)},
	}})

	require.Error(t, err)
	assert.Contains(t, describe(err), `service name "overview" is taken by a dashboard`)
}

func TestGenerate_ServiceDashboardReadsLogsOfTheSelectedProcesses(t *testing.T) {
	_, orders := generateOrders(t)

	queries := map[string]string{}

	for _, panel := range orders.section(t, "Logs").panels {
		for _, target := range panel.Targets {
			queries[panel.Title] = target.Expr
		}
	}

	assert.Equal(t, map[string]string{
		"Volume por nível": `sum by (detected_level) (count_over_time({service_namespace="boilerplate", ` +
			`service_name=~"$process"} [$__auto]))`,
		"Logs de erro": `{service_namespace="boilerplate", service_name=~"$process"} | ` +
			`detected_level=~"error|critical|fatal"`,
	}, queries)
}
