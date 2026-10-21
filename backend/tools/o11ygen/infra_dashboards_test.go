package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/dashboards"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const observabilityCompose = "../../docker-compose.observability.yml"

func panelsOf(board map[string]any) []map[string]any {
	var out []map[string]any

	var walk func(list []any)

	walk = func(list []any) {
		for _, p := range list {
			panel := p.(map[string]any)
			out = append(out, panel)

			if nested, ok := panel["panels"].([]any); ok {
				walk(nested)
			}
		}
	}

	walk(board["panels"].([]any))

	return out
}

func exprsOf(board map[string]any) []string {
	var exprs []string

	for _, panel := range panelsOf(board) {
		targets, _ := panel["targets"].([]any)
		for _, target := range targets {
			if expr, ok := target.(map[string]any)["expr"].(string); ok {
				exprs = append(exprs, expr)
			}
		}
	}

	return exprs
}

func alertListFilters(board map[string]any) []string {
	var filters []string

	for _, panel := range panelsOf(board) {
		if panel["type"] == "alertlist" {
			filters = append(filters, panel["options"].(map[string]any)["alertInstanceLabelFilter"].(string))
		}
	}

	return filters
}

func linkURLs(board map[string]any) []string {
	var urls []string

	for _, link := range board["links"].([]any) {
		if url, ok := link.(map[string]any)["url"].(string); ok && url != "" {
			urls = append(urls, url)
		}
	}

	return urls
}

func TestGenerate_KeepsTheInfraDashboardsAndTheHomeValidWithoutServices(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, nil))

	kind := dashboards.Kind()

	for name, uid := range map[string]string{
		"kafka.json":    "boilerplate-kafka",
		"postgres.json": "boilerplate-postgres",
		"overview.json": "boilerplate-overview",
	} {
		assert.Equal(t, uid, readDashboard(t, root, name)["uid"], name)

		content := readFile(t, filepath.Join(root, dashboardsDir, name))

		queries, err := kind.Queries(content)
		require.NoError(t, err, describe(err))
		assert.NotEmpty(t, queries, name)

		for _, query := range queries {
			assert.NoError(t, catalog.Check(query), "%s: %s: %s", name, query.Origin, query.Expr)
		}
	}
}

func TestGenerate_ListsTheMessagingAlertsOnTheKafkaDashboard(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, nil))

	assert.Equal(t, []string{`{alertname=~"ConsumerLagHigh|DLQMessageReceived|KafkaBrokersDown"}`},
		alertListFilters(readDashboard(t, root, "kafka.json")))
}

func TestGenerate_ListsThePostgresAndCardinalityAlertsOnThePostgresDashboard(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, nil))

	assert.Equal(t, []string{`{alertname=~"SeriesBudgetExceeded|PostgresDown|PostgresConnectionsNearMax"}`},
		alertListFilters(readDashboard(t, root, "postgres.json")))
}

func TestGenerate_PointsTheInfraAndCardinalityAlertsAtTheInfraDashboards(t *testing.T) {
	root := t.TempDir()
	declared := []service.Service{{Name: "orders", Processes: []service.Process{service.Daemon("orders-api", service.API)}}}

	require.NoError(t, generate(root, declared))

	urls := map[string]string{}

	for _, file := range []string{"infra.yaml", "orders.yaml"} {
		for _, group := range readRules(t, filepath.Join(root, rulesDir, file)).Groups {
			for _, rule := range group.Rules {
				if rule.Alert != "" {
					urls[rule.Alert] = rule.Annotations["dashboard_url"]
				}
			}
		}
	}

	for alert, want := range map[string]string{
		"KafkaBrokersDown":           "http://localhost:3010/d/boilerplate-kafka",
		"PostgresDown":               "http://localhost:3010/d/boilerplate-postgres",
		"PostgresConnectionsNearMax": "http://localhost:3010/d/boilerplate-postgres",
		"SeriesBudgetExceeded":       "http://localhost:3010/d/boilerplate-postgres",
	} {
		assert.Equal(t, want, urls[alert], alert)
	}
}

func TestGenerate_RanksTheMetricsBySeriesOnThePostgresDashboard(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, nil))

	assert.Contains(t, exprsOf(readDashboard(t, root, "postgres.json")),
		`topk(20, count by (__name__) ({__name__=~".+", job=~"$job"}))`)
}

func TestGenerate_OverviewShowsTheSLOsAlertsServiceMapAndInfraLinksInsteadOfRED(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	board := readDashboard(t, root, "overview.json")

	panels := panelsOf(board)

	types := make([]string, 0, len(panels))
	for _, panel := range panels {
		types = append(types, panel["type"].(string))
	}

	assert.ElementsMatch(t, []string{"table", "alertlist", "nodeGraph"}, types)
	assert.Equal(t, []string{""}, alertListFilters(board))
	assert.Equal(t, []string{"/d/boilerplate-kafka", "/d/boilerplate-postgres"}, linkURLs(board))

	exprs := strings.Join(exprsOf(board), "\n")
	assert.Contains(t, exprs, catalog.SLOPeriodErrorBudgetRemainingRatio)
	assert.NotContains(t, exprs, catalog.HTTPServerRequestTotal)
}

func TestGenerate_OverviewIsTheGrafanaHome(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, nil))

	compose := readFile(t, observabilityCompose)

	home := regexp.MustCompile(`GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH=/etc/grafana/provisioning/dashboards/(\S+)`).
		FindSubmatch(compose)
	require.NotNil(t, home, "Grafana has no default_home_dashboard_path")

	assert.Equal(t, "boilerplate-overview", readDashboard(t, root, string(home[1]))["uid"])
}
