package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		"http.json":         "boilerplate-http",
		"logs.json":         "boilerplate-logs",
		"messaging.json":    "boilerplate-messaging",
		"notes-outbox.json": "boilerplate-notes-outbox",
		"overview.json":     "boilerplate-overview",
		"traces.json":       "boilerplate-traces",
	}, uids)
}

func TestGenerate_ReferencesDatasourcesThroughVariables(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	board := readDashboard(t, root, "traces.json")

	uids := map[string]bool{}
	collectDatasourceUIDs(board, uids)

	assert.Equal(t, map[string]bool{"-- Grafana --": true, "${loki}": true, "${tempo}": true}, uids)

	var variables []string

	for _, v := range board["templating"].(map[string]any)["list"].([]any) {
		variable := v.(map[string]any)
		if variable["type"] == "datasource" {
			variables = append(variables, variable["name"].(string)+"="+variable["query"].(string))
		}
	}

	assert.Equal(t, []string{"loki=loki", "tempo=tempo"}, variables)
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
