package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func collectDatasourceUIDs(node any, uids map[string]bool) {
	switch v := node.(type) {
	case map[string]any:
		if ds, ok := v["datasource"].(map[string]any); ok {
			if uid, ok := ds["uid"].(string); ok {
				uids[uid] = true
			}
		}

		for _, child := range v {
			collectDatasourceUIDs(child, uids)
		}
	case []any:
		for _, child := range v {
			collectDatasourceUIDs(child, uids)
		}
	}
}

func TestGenerate_WritesTheDashboards(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	entries, err := os.ReadDir(filepath.Join(root, dashboardsDir))
	require.NoError(t, err)

	uids := map[string]string{}

	for _, entry := range entries {
		uids[entry.Name()], _ = readDashboard(t, root, entry.Name())["uid"].(string)
	}

	assert.Equal(t, map[string]string{
		"http.json":  "boilerplate-http",
		"kafka.json": "boilerplate-kafka",
		"logs.json":  "boilerplate-logs",
		// example:begin
		"notes-outbox.json": "boilerplate-notes-outbox",
		// example:end
		"overview.json": "boilerplate-overview",
		"postgres.json": "boilerplate-postgres",
		"traces.json":   "boilerplate-traces",
	}, uids)
}

func TestGenerate_ReferencesDatasourcesThroughVariables(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	board := readDashboard(t, root, "traces.json")

	uids := map[string]bool{}
	collectDatasourceUIDs(board, uids)

	assert.Equal(t, map[string]bool{"-- Grafana --": true, "${loki_datasource}": true, "${tempo_datasource}": true}, uids)

	var variables []string

	for _, v := range board["templating"].(map[string]any)["list"].([]any) {
		variable := v.(map[string]any)
		if variable["type"] == "datasource" {
			variables = append(variables, variable["name"].(string)+"="+variable["query"].(string))
		}
	}

	assert.Equal(t, []string{"tempo_datasource=tempo", "loki_datasource=loki"}, variables)
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

	for _, kind := range kinds() {
		require.NotNil(t, kind.Queries, "kind %s/%s registers no query extractor", kind.Dir, kind.Glob)

		paths, err := filepath.Glob(filepath.Join(root, kind.Dir, kind.Glob))
		require.NoError(t, err)

		for _, path := range paths {
			content, err := os.ReadFile(path) //nolint:gosec // The path comes from globbing the test's temporary directory.
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
	for _, kind := range kinds() {
		assert.NotZero(t, kind.LoadedBy, "kind %s/%s does not say what loads it, so check-live cannot verify it", kind.Dir, kind.Glob)
	}
}

func describe(err error) string {
	if err == nil {
		return ""
	}

	return strings.Join(append([]string{err.Error()}, errors.Reasons(err)...), ": ")
}
