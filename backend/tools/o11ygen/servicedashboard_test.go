package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

type boardTarget struct {
	Expr      string `json:"expr"`
	Legend    string `json:"legendFormat"` //nolint:tagliatelle // Grafana's dashboard schema.
	Exemplar  bool   `json:"exemplar"`
	QueryType string `json:"queryType"` //nolint:tagliatelle // Grafana's dashboard schema.
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

	raw := readFile(t, filepath.Join(root, dashboardsDir, name))

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
		"HealthCheckFailing":     ordersBoard,
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
		"query environment",
		"custom process",
	}, variables)
}

func TestGenerate_ServiceDashboardTopsWithTheSlothSLOsAndTheServiceAlerts(t *testing.T) {
	_, orders := generateOrders(t)

	top := orders.section(t, "SLOs e alertas")

	queries := map[string][]string{}
	filter := ""

	for _, panel := range top.panels {
		for _, target := range panel.Targets {
			queries[panel.Type+" "+panel.Title] = append(queries[panel.Type+" "+panel.Title], target.Expr)
		}

		if panel.Type == "alertlist" {
			filter, _ = panel.Options["alertInstanceLabelFilter"].(string)
		}
	}

	bySLO := "max by (sloth_service, sloth_slo, service) "
	inEnvironment := `{sloth_service="orders", deployment_environment="$environment"})`
	assert.Equal(t, map[string][]string{"table SLOs": {
		bySLO + `(slo:objective:ratio{sloth_service="orders"})`,
		"1 - " + bySLO + `(slo:sli_error:ratio_rate30d` + inEnvironment,
		bySLO + `(slo:period_error_budget_remaining:ratio` + inEnvironment,
		bySLO + `(slo:current_burn_rate:ratio` + inEnvironment,
	}}, queries)
	assert.Equal(t, `{service=~"orders-api|orders-consumer|orders-worker", deployment_environment=~"$environment|"}`, filter)
}

func sloTableShape(t *testing.T, board map[string]any) map[string]any {
	t.Helper()

	for _, panel := range panelsOf(board) {
		if panel["type"] != "table" {
			continue
		}

		shape := map[string]any{}

		for key, value := range panel {
			if key != "id" && key != "gridPos" && key != "targets" {
				shape[key] = value
			}
		}

		return shape
	}

	require.Fail(t, "no SLO table")

	return nil
}

func TestGenerate_ServiceDashboardShowsItsSLOsInTheOverviewTable(t *testing.T) {
	root, _ := generateOrders(t)

	assert.Equal(t, sloTableShape(t, readDashboard(t, root, "overview.json")),
		sloTableShape(t, readDashboard(t, root, "orders.json")))
}

func exemplarExprs(panels []boardPanel) []string {
	var out []string

	for i := range panels {
		for _, target := range panels[i].Targets {
			if target.Exemplar {
				out = append(out, panels[i].Title+": "+target.Expr)
			}
		}

		out = append(out, exemplarExprs(panels[i].Panels)...)
	}

	return out
}

func TestGenerate_DashboardsAskForExemplarsOnlyOnTheGoLatencyHistograms(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{ordersService, storefrontService}))

	files, err := filepath.Glob(filepath.Join(root, dashboardsDir, "*.json"))
	require.NoError(t, err)

	exemplars := map[string][]string{}

	for _, file := range files {
		name := filepath.Base(file)
		if exprs := exemplarExprs(readBoard(t, root, name).Panels); exprs != nil {
			exemplars[name] = exprs
		}
	}

	api := `{job=~"boilerplate/(orders-api)", deployment_environment="$environment", http_route!~"/ping|/health"}[$__rate_interval])))`
	byRoute := "le, http_response_status_code, http_request_method, http_route"
	consumer := `{job=~"boilerplate/(orders-consumer)", deployment_environment="$environment"}[$__rate_interval])))`
	worker := `{job=~"boilerplate/(orders-worker)", deployment_environment="$environment"}[$__rate_interval])))`

	want := make([]string, 0, 12)

	for _, family := range []struct{ title, by, histogram, scope string }{
		{"Latência %s", "le", "http_server_request_duration_seconds", api},
		{"Latência %s por rota", byRoute, "http_server_request_duration_seconds", api},
		{"Duração do processamento %s", "le", "messaging_process_duration_seconds", consumer},
		{"Duração do envio %s", "le", "messaging_client_operation_duration_seconds", worker},
	} {
		for _, p := range []struct{ name, q string }{{"p50", "0.5"}, {"p90", "0.9"}, {"p99", "0.99"}} {
			want = append(want, fmt.Sprintf(family.title, p.name)+": histogram_quantile("+p.q+", sum by ("+family.by+
				") (rate("+family.histogram+"_bucket"+family.scope)
		}
	}

	assert.Equal(t, map[string][]string{"orders.json": want}, exemplars)
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
		"Latência p50":                  0.25,
		"Latência p90":                  0.25,
		"Latência p99":                  0.25,
		"Latência p50 por rota":         0.25,
		"Latência p90 por rota":         0.25,
		"Latência p99 por rota":         0.25,
		"Duração do processamento p50":  1,
		"Duração do processamento p90":  1,
		"Duração do processamento p99":  1,
		"Idade da pendente mais antiga": 30,
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
		return "/a/grafana-lokiexplore-app/explore/service_name/" + process + "/logs?var-ds=loki" +
			"&var-filters=service_name%7C%3D%7C" + process
	}
	profiles := func(process string) string {
		return "/a/grafana-pyroscope-app/explore?explorationType=flame-graph&var-serviceName=" + process +
			"&var-profileMetricId=process_cpu%3Acpu%3Ananoseconds%3Acpu%3Ananoseconds"
	}

	assert.Equal(t, map[string]string{
		"Traces": "/a/grafana-exploretraces-app/explore?var-ds=tempo" +
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

	require.Len(t, links, 3)
	assert.Equal(t, links[0], links[1])
	assert.Equal(t, links[0], links[2])
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

	assert.Equal(t, "tempo", panes["slow"].Datasource)
	assert.Equal(t, "traceql", panes["slow"].Queries[0].QueryType)
	assert.Equal(t, `{resource.service.name=~"orders-api" && kind=server && span.http.route="${__field.labels.http_route}" `+
		`&& span.http.request.method="${__field.labels.http_request_method}" `+
		`&& span.http.response.status_code=${__field.labels.http_response_status_code} `+
		`&& duration > 250ms}`, panes["slow"].Queries[0].Query)
	assert.Equal(t, map[string]string{"from": "${__from}", "to": "${__to}"}, panes["slow"].Range)
}

func legendsByTitle(panels []boardPanel, titles ...string) map[string][]string {
	out := map[string][]string{}

	for i := range panels {
		for _, title := range titles {
			if panels[i].Title != title {
				continue
			}

			for _, target := range panels[i].Targets {
				out[title] = append(out[title], target.Expr+" => "+target.Legend)
			}
		}
	}

	return out
}

func TestGenerate_ServiceDashboardGroupsEveryRoutePanelByStatusMethodAndRoute(t *testing.T) {
	_, orders := generateOrders(t)

	selector := `job=~"boilerplate/(orders-api)", deployment_environment="$environment", http_route!~"/ping|/health"`
	byRoute := "http_response_status_code, http_request_method, http_route"
	legend := "{{http_response_status_code}} {{http_request_method}} {{http_route}}"
	quantile := func(q string) string {
		return "histogram_quantile(" + q + ", sum by (le, " + byRoute + ") (rate(http_server_request_duration_seconds_bucket{" +
			selector + "}[$__rate_interval])))"
	}

	assert.Equal(t, map[string][]string{
		"Taxa por rota": {
			"sum by (" + byRoute + ") (rate(http_server_request_total{" + selector + "}[$__rate_interval])) => " + legend,
		},
		"4xx/5xx por rota": {
			"sum by (" + byRoute + ") (rate(http_server_request_total{" + selector +
				`, http_response_status_code=~"4..|5.."}[$__rate_interval])) => ` + legend,
		},
		"Latência p50 por rota": {quantile("0.5") + " => p50 " + legend},
		"Latência p90 por rota": {quantile("0.9") + " => p90 " + legend},
		"Latência p99 por rota": {quantile("0.99") + " => p99 " + legend},
	}, legendsByTitle(orders.section(t, "api").panels, "Taxa por rota", "4xx/5xx por rota",
		"Latência p50 por rota", "Latência p90 por rota", "Latência p99 por rota"))
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
			`service_name=~"$process", deployment_environment="$environment"} [$__auto]))`,
		"Logs de erro": `{service_namespace="boilerplate", service_name=~"$process", deployment_environment="$environment"} | ` +
			`detected_level=~"error|critical|fatal"`,
	}, queries)
}

func TestGenerate_ServiceDashboardShowsTheCPUAndRSSOfEachProcessInTheRuntimeSection(t *testing.T) {
	_, orders := generateOrders(t)

	runtime := orders.section(t, "Runtime")
	titles := make([]string, 0, len(runtime.panels))
	queries := map[string][]string{}

	for _, panel := range runtime.panels {
		titles = append(titles, panel.Title)

		for _, target := range panel.Targets {
			queries[panel.Title] = append(queries[panel.Title], target.Expr)
		}
	}

	assert.Equal(t, []string{"CPU", "RSS", "Goroutines", "Memória Go", "GC"}, titles)
	assert.Equal(t, []string{
		`sum by (job, cpu_mode) (rate(process_cpu_time_seconds_total{job=~"boilerplate/$process", deployment_environment="$environment"}[$__rate_interval]))`,
	}, queries["CPU"])
	assert.Equal(t, []string{`sum by (job) (process_memory_usage_bytes{job=~"boilerplate/$process", deployment_environment="$environment"})`}, queries["RSS"])
}
