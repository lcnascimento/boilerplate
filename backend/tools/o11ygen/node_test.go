package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

var storefrontService = service.Service{
	Name:      "storefront",
	Runtime:   service.Node,
	Processes: []service.Process{service.Daemon("storefront", service.API, service.Browser)},
}

func generateStorefront(t *testing.T) (string, board) {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{storefrontService}))

	return root, readBoard(t, root, "storefront.json")
}

func exprsByTitle(panels []boardPanel) map[string][]string {
	out := map[string][]string{}

	for i := range panels {
		for _, target := range panels[i].Targets {
			out[panels[i].Title] = append(out[panels[i].Title], target.Expr)
		}
	}

	return out
}

func TestGenerate_NodeServiceReadsItsAPIFromTheServerSpanMetrics(t *testing.T) {
	_, storefront := generateStorefront(t)

	server := `service=~"storefront", span_kind="SPAN_KIND_SERVER"`
	calls := func(extra, by string) string {
		return "sum" + by + " (rate(traces_spanmetrics_calls_total{" + server + extra + "}[$__rate_interval]))"
	}
	quantile := func(q, by string) string {
		return "histogram_quantile(" + q + ", sum by (" + by + ") (rate(traces_spanmetrics_latency_bucket{" + server +
			"}[$__rate_interval])))"
	}
	failed := `, status_code="STATUS_CODE_ERROR"`

	assert.Equal(t, map[string][]string{
		"Taxa":                  {calls("", "")},
		"Erros":                 {"(" + calls(failed, "") + " or vector(0)) / " + calls("", "")},
		"p99":                   {quantile("0.99", "le")},
		"Taxa por rota":         {calls("", " by (span_name)")},
		"Erros por rota":        {calls(failed, " by (span_name)")},
		"Latência p50/p99":      {quantile("0.5", "le"), quantile("0.99", "le")},
		"Latência p99 por rota": {quantile("0.99", "le, span_name")},
	}, exprsByTitle(storefront.section(t, "api").panels))
}

func TestGenerate_NodeServiceShowsTheNodeRuntime(t *testing.T) {
	_, storefront := generateStorefront(t)

	sections := storefront.sections()
	rows := make([]string, 0, len(sections))

	for i := range sections {
		rows = append(rows, sections[i].row.Title)
	}

	assert.Equal(t, []string{"SLOs e alertas", "api", "Runtime", "Logs"}, rows)

	job := `{job=~"boilerplate/$process"}`

	assert.Equal(t, map[string][]string{
		"Event loop": {
			"max by (job) (nodejs_eventloop_delay_p99_seconds" + job + ")",
		},
		"Utilização do event loop": {
			"max by (job) (nodejs_eventloop_utilization_ratio" + job + ")",
		},
		"Heap": {
			"sum by (job) (v8js_memory_heap_used_bytes" + job + ")",
		},
		"GC": {
			"sum by (job, v8js_gc_type) (rate(v8js_gc_duration_seconds_sum" + job + "[$__rate_interval]))",
		},
	}, exprsByTitle(storefront.section(t, "Runtime").panels))
}

func TestGenerate_NodeServiceInheritsNoSLOButTheLongRunningProcessAlerts(t *testing.T) {
	root, storefront := generateStorefront(t)

	for _, dir := range []string{"config/sloth", sloRulesDir} {
		_, err := os.Stat(filepath.Join(root, dir, "storefront.yaml"))
		assert.ErrorIs(t, err, os.ErrNotExist, dir)
	}

	assert.Equal(t, map[string]map[string][]string{
		"storefront": {"storefront": {"ServiceTelemetryAbsent", "SeriesBudgetExceeded", "CriticalLogRecorded"}},
	}, alertsByGroupAndService(readRules(t, filepath.Join(root, rulesDir, "storefront.yaml"))))

	for _, link := range storefront.Links {
		assert.False(t, strings.HasPrefix(link.Title, "Profiles"), "a Node process sends no profile: %s", link.Title)
	}
}

func TestGenerate_RejectsANodeProcessThatRunsWhatOnlyGoKitInstruments(t *testing.T) {
	for name, process := range map[string]service.Process{
		"consumer": service.Daemon("storefront", service.Consumer).WithConsumerGroup("storefront"),
		"worker":   service.Daemon("storefront", service.Worker),
		"db pool":  service.Daemon("storefront", service.API).WithDBPool(),
	} {
		t.Run(name, func(t *testing.T) {
			err := generate(t.TempDir(), []service.Service{{
				Name: "storefront", Runtime: service.Node, Processes: []service.Process{process},
			}})

			require.ErrorContains(t, err, `process "storefront" runs on Node, which only instruments api and browser`)
		})
	}
}
