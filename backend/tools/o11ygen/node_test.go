package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

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
		"Taxa por rota":         {calls("", " by (http_status_code, span_name)")},
		"Erros por rota":        {calls(failed, " by (http_status_code, span_name)")},
		"Latência p50":          {quantile("0.5", "le")},
		"Latência p90":          {quantile("0.9", "le")},
		"Latência p99":          {quantile("0.99", "le")},
		"Latência p50 por rota": {quantile("0.5", "le, http_status_code, span_name")},
		"Latência p90 por rota": {quantile("0.9", "le, http_status_code, span_name")},
		"Latência p99 por rota": {quantile("0.99", "le, http_status_code, span_name")},
	}, exprsByTitle(storefront.section(t, "api").panels))
}

func TestGenerate_NodeServiceLabelsEveryRouteSeriesWithStatusMethodAndRoute(t *testing.T) {
	_, storefront := generateStorefront(t)

	legends := map[string][]string{}

	for _, panel := range storefront.section(t, "api").panels {
		for _, target := range panel.Targets {
			if target.Legend != "" {
				legends[panel.Title] = append(legends[panel.Title], target.Legend)
			}
		}
	}

	assert.Equal(t, map[string][]string{
		"Taxa por rota":         {"{{http_status_code}} {{span_name}}"},
		"Erros por rota":        {"{{http_status_code}} {{span_name}}"},
		"Latência p50":          {"p50"},
		"Latência p90":          {"p90"},
		"Latência p99":          {"p99"},
		"Latência p50 por rota": {"p50 {{http_status_code}} {{span_name}}"},
		"Latência p90 por rota": {"p90 {{http_status_code}} {{span_name}}"},
		"Latência p99 por rota": {"p99 {{http_status_code}} {{span_name}}"},
	}, legends)
}

func TestTempoKeepsTheHTTPStatusOfEverySpanMetric(t *testing.T) {
	raw, err := os.ReadFile("../../config/tempo/tempo.yaml")
	require.NoError(t, err)

	var config struct {
		MetricsGenerator struct {
			Processor struct {
				SpanMetrics struct {
					Dimensions []string `yaml:"dimensions"`
				} `yaml:"span_metrics"`
			} `yaml:"processor"`
		} `yaml:"metrics_generator"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &config))

	assert.Contains(t, config.MetricsGenerator.Processor.SpanMetrics.Dimensions, "http.status_code")
}

func TestGenerate_NodeServiceShowsTheNodeRuntime(t *testing.T) {
	_, storefront := generateStorefront(t)

	sections := storefront.sections()
	rows := make([]string, 0, len(sections))

	for i := range sections {
		rows = append(rows, sections[i].row.Title)
	}

	assert.Equal(t, []string{"Alertas", "api", "browser", "Runtime", "Logs"}, rows)

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

func TestGenerate_ServiceWithoutSLOsTopsWithItsAlertsOnly(t *testing.T) {
	_, storefront := generateStorefront(t)

	top := storefront.section(t, "Alertas")

	require.Len(t, top.panels, 1)
	assert.Equal(t, "alertlist", top.panels[0].Type)
	assert.Equal(t, `{service=~"storefront"}`, top.panels[0].Options["alertInstanceLabelFilter"])
}

func TestGenerate_BrowserSectionReadsTheFaroWebVitalsErrorsAndPageLoadsFromLoki(t *testing.T) {
	_, storefront := generateStorefront(t)

	stream := `{service_namespace="boilerplate", service_name=~"storefront"}`
	p75 := func(vital string) string {
		return `quantile_over_time(0.75, ` + stream + ` | kind="measurement" | type="web-vitals" | value_` + vital +
			`!="" | unwrap value_` + vital + ` [$__auto]) by ()`
	}
	pageLoads := `sum(count_over_time(` + stream + ` | kind="event" | event_name="faro.performance.navigation" [$__auto]))`
	jsErrors := func(by string) string {
		return `sum` + by + `(count_over_time(` + stream + ` | kind="exception" [$__auto]))`
	}

	browser := storefront.section(t, "browser").panels

	assert.Equal(t, map[string][]string{
		"LCP":                       {p75("lcp")},
		"INP":                       {p75("inp")},
		"CLS":                       {p75("cls")},
		"FCP":                       {p75("fcp")},
		"TTFB":                      {p75("ttfb")},
		"Erros de JS por page load": {"(" + jsErrors("") + " or vector(0)) / " + pageLoads},
		"Web Vitals de tempo (p75)": {p75("lcp"), p75("inp"), p75("fcp"), p75("ttfb")},
		"CLS (p75)":                 {p75("cls")},
		"Page loads":                {pageLoads},
		"Erros de JS por tipo":      {jsErrors(" by (type) ")},
	}, exprsByTitle(browser))

	kinds := map[string]string{}

	for i := range browser {
		for _, target := range browser[i].Targets {
			kinds[browser[i].Title] = browser[i].Type + " " + target.QueryType
		}
	}

	assert.Equal(t, map[string]string{
		"LCP": "stat instant", "INP": "stat instant", "CLS": "stat instant", "FCP": "stat instant",
		"TTFB": "stat instant", "Erros de JS por page load": "stat instant",
		"Web Vitals de tempo (p75)": "timeseries ", "CLS (p75)": "timeseries ", "Page loads": "timeseries ",
		"Erros de JS por tipo": "timeseries ",
	}, kinds)
}
