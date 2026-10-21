package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const dashboardsDir = "config/grafana/provisioning/dashboards"

func readDashboard(t *testing.T, root, name string) map[string]any {
	t.Helper()

	raw, err := fs.ReadFile(os.DirFS(root), path.Join(dashboardsDir, name))
	require.NoError(t, err)

	var board map[string]any
	require.NoError(t, json.Unmarshal(raw, &board))

	return board
}

func collectDatasources(node any, refs map[string]bool) {
	switch v := node.(type) {
	case map[string]any:
		if ds, ok := v["datasource"].(map[string]any); ok {
			kind, _ := ds["type"].(string)
			uid, _ := ds["uid"].(string)
			refs[kind+"/"+uid] = true
		}

		for _, child := range v {
			collectDatasources(child, refs)
		}
	case []any:
		for _, child := range v {
			collectDatasources(child, refs)
		}
	}
}

func TestGenerate_WritesTheDashboardsTitledByTheirLowercaseName(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	entries, err := os.ReadDir(filepath.Join(root, dashboardsDir))
	require.NoError(t, err)

	boards := map[string]string{}

	for _, entry := range entries {
		board := readDashboard(t, root, entry.Name())
		boards[entry.Name()] = board["title"].(string) + " " + board["uid"].(string)
	}

	assert.Equal(t, map[string]string{
		"kafka.json":    "kafka boilerplate-kafka",
		"overview.json": "overview boilerplate-overview",
		"postgres.json": "postgres boilerplate-postgres",
		"web.json":      "web boilerplate-web",
		// example:begin
		"audit.json": "audit boilerplate-audit",
		"notes.json": "notes boilerplate-notes",
		// example:end
	}, boards)
}

func provisioned(t *testing.T) map[string]bool {
	t.Helper()

	raw, err := os.ReadFile("../../config/grafana/provisioning/datasources/datasources.yaml")
	require.NoError(t, err)

	var provisioning struct {
		Datasources []struct {
			Type string `yaml:"type"`
			UID  string `yaml:"uid"`
		} `yaml:"datasources"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &provisioning))

	out := map[string]bool{"grafana/-- Grafana --": true}
	for _, ds := range provisioning.Datasources {
		out[ds.Type+"/"+ds.UID] = true
	}

	return out
}

func TestGenerate_ReferencesTheProvisionedDatasourcesByUIDWithoutDatasourceVariables(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{ordersService, storefrontService}))

	files, err := filepath.Glob(filepath.Join(root, dashboardsDir, "*.json"))
	require.NoError(t, err)

	allowed := provisioned(t)
	used := map[string]bool{}

	for _, file := range files {
		board := readDashboard(t, root, filepath.Base(file))

		refs := map[string]bool{}
		collectDatasources(board, refs)

		for ref := range refs {
			assert.True(t, allowed[ref], "%s references %q, which is not provisioned", filepath.Base(file), ref)
			used[ref] = true
		}

		for _, v := range readBoard(t, root, filepath.Base(file)).Templating.List {
			assert.NotEqual(t, "datasource", v.Type, filepath.Base(file)+" "+v.Name)
		}
	}

	assert.Equal(t, map[string]bool{
		"grafana/-- Grafana --": true, "prometheus/prometheus": true, "loki/loki": true, "tempo/tempo": true,
	}, used)
}

func TestGenerate_RemovesStaleDashboardsAndKeepsTheProvider(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, dashboardsDir)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "removed-service.json"), []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dashboards.yaml"), []byte("apiVersion: 1\n"), 0o600))

	require.NoError(t, generate(root, services))

	assert.NoFileExists(t, filepath.Join(dir, "removed-service.json"))
	assert.FileExists(t, filepath.Join(dir, "dashboards.yaml"))
}

func TestGenerate_RejectsAProcessDeclaredTwice(t *testing.T) {
	declared := []service.Service{
		{Name: "orders", Processes: []service.Process{service.Daemon("orders-api", service.API)}},
		{Name: "billing", Processes: []service.Process{service.Daemon("orders-api", service.Consumer)}},
	}

	err := generate(t.TempDir(), declared)

	require.ErrorContains(t, err, `process "orders-api" is declared twice`)
}

func TestGenerate_QueriesStayInsideTheCatalog(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	checked := 0

	for _, kind := range loadedKinds() {
		require.NotNil(t, kind.Queries, "kind %s/%s registers no query extractor", kind.Dir, kind.Glob)

		paths, err := filepath.Glob(filepath.Join(root, kind.Dir, kind.Glob))
		require.NoError(t, err)

		for _, path := range paths {
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			queries, err := kind.Queries(content)
			require.NoError(t, err, describe(err))

			for _, query := range queries {
				checked++

				if err := catalog.Check(query); err != nil {
					t.Errorf("%s/%s: %s: %s\n  %s", kind.Dir, filepath.Base(path), query.Origin, describe(err), query.Expr)
				}
			}
		}
	}

	assert.Positive(t, checked, "no generated query was checked")
}

func TestKinds_NameWhatLoadsThem(t *testing.T) {
	for _, kind := range loadedKinds() {
		assert.NotZero(t, kind.LoadedBy, "kind %s/%s does not say what loads it, so check-live cannot verify it", kind.Dir, kind.Glob)
	}
}

func describe(err error) string {
	if err == nil {
		return ""
	}

	return strings.Join(append([]string{err.Error()}, errors.Reasons(err)...), ": ")
}

func collectPromQL(node any, out *[]string) {
	switch v := node.(type) {
	case map[string]any:
		if expr, ok := v["expr"].(string); ok {
			if ds, ok := v["datasource"].(map[string]any); ok && ds["type"] == "prometheus" {
				*out = append(*out, expr)
			}
		}

		for _, child := range v {
			collectPromQL(child, out)
		}
	case []any:
		for _, child := range v {
			collectPromQL(child, out)
		}
	}
}

var (
	rateCall  = regexp.MustCompile(`\bi?rate\(`)
	rateRange = regexp.MustCompile(`\bi?rate\([a-zA-Z_:][\w:]*(?:\{(?:[^"}]|"(?:[^"\\]|\\.)*")*\})?\[([^\]]+)\]`)
)

// Stands in for dashboard-linter's target-rate-interval-rule, which only runs on a dashboard with a Prometheus
// datasource variable.
func TestGenerate_RatesEveryDashboardQueryOverTheRateInterval(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{ordersService, storefrontService}))

	files, err := filepath.Glob(filepath.Join(root, dashboardsDir, "*.json"))
	require.NoError(t, err)

	var exprs []string
	for _, file := range files {
		collectPromQL(readDashboard(t, root, filepath.Base(file)), &exprs)
	}

	require.NotEmpty(t, exprs)

	for _, expr := range exprs {
		ranges := rateRange.FindAllStringSubmatch(expr, -1)
		assert.Len(t, ranges, len(rateCall.FindAllString(expr, -1)), "a rate() without a plain range selector: %s", expr)

		for _, r := range ranges {
			assert.Equal(t, "$__rate_interval", r[1], expr)
		}
	}
}
