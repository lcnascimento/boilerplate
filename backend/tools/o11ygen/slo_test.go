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

const (
	sloSpecsDir = "config/sloth"
	sloRulesDir = "config/prometheus/slo"
)

type sloSpec struct {
	Version    string `yaml:"version"`
	Service    string `yaml:"service"`
	SLOPlugins struct {
		Chain []struct {
			ID     string         `yaml:"id"`
			Config map[string]any `yaml:"config"`
		} `yaml:"chain"`
	} `yaml:"slo_plugins"`
	SLOs []struct {
		Name      string            `yaml:"name"`
		Objective float64           `yaml:"objective"`
		Labels    map[string]string `yaml:"labels"`
		SLI       struct {
			Events *struct {
				ErrorQuery string `yaml:"error_query"`
				TotalQuery string `yaml:"total_query"`
			} `yaml:"events"`
			Raw *struct {
				ErrorRatioQuery string `yaml:"error_ratio_query"`
			} `yaml:"raw"`
		} `yaml:"sli"`
		Alerting struct {
			Name        string            `yaml:"name"`
			Labels      map[string]string `yaml:"labels"`
			Annotations map[string]string `yaml:"annotations"`
			PageAlert   struct {
				Labels map[string]string `yaml:"labels"`
			} `yaml:"page_alert"`
			TicketAlert struct {
				Labels map[string]string `yaml:"labels"`
			} `yaml:"ticket_alert"`
		} `yaml:"alerting"`
	} `yaml:"slos"`
}

func readSLOSpec(t *testing.T, root, svc string) sloSpec {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, sloSpecsDir, svc+".yaml")) //nolint:gosec // Test temporary directory.
	require.NoError(t, err)

	var spec sloSpec
	require.NoError(t, yaml.Unmarshal(raw, &spec))

	return spec
}

var ordersService = service.Service{
	Name: "orders",
	Processes: []service.Process{
		service.Daemon("orders-api", service.API).WithDBPool(),
		service.Daemon("orders-consumer", service.Consumer).WithConsumerGroup("orders"),
		service.Daemon("orders-worker", service.Worker),
		service.OneShot("orders-migrate"),
	},
}

func TestGenerate_WritesASlothSpecWithTheDefaultSLOsOfEachComponent(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, []service.Service{ordersService}))

	spec := readSLOSpec(t, root, "orders")

	assert.Equal(t, "prometheus/v1", spec.Version)
	assert.Equal(t, "orders", spec.Service)
	require.Len(t, spec.SLOPlugins.Chain, 1)
	assert.Equal(t, "sloth.dev/core/sli_rules/v1", spec.SLOPlugins.Chain[0].ID)
	assert.Equal(t, map[string]any{"disableOptimized": true}, spec.SLOPlugins.Chain[0].Config)

	type slo struct {
		service, errorQuery, totalQuery, errorRatioQuery string
		objective                                        float64
	}

	got := map[string]slo{}

	for _, s := range spec.SLOs {
		entry := slo{service: s.Labels["service"], objective: s.Objective}
		if s.SLI.Events != nil {
			entry.errorQuery, entry.totalQuery = s.SLI.Events.ErrorQuery, s.SLI.Events.TotalQuery
		}

		if s.SLI.Raw != nil {
			entry.errorRatioQuery = s.SLI.Raw.ErrorRatioQuery
		}

		got[s.Name] = entry
	}

	const (
		api      = `job="boilerplate/orders-api",http_route!~"/ping|/health"`
		consumer = `job="boilerplate/orders-consumer"`
	)

	assert.Equal(t, map[string]slo{
		"api-availability": {
			service: "orders-api", objective: 99.9,
			errorQuery: `sum(rate(http_server_request_total{` + api + `,http_response_status_code=~"5.."}[{{.window}}])) or vector(0)`,
			totalQuery: `sum(rate(http_server_request_total{` + api + `}[{{.window}}]))`,
		},
		"api-latency": {
			service: "orders-api", objective: 99,
			errorQuery: `sum(rate(http_server_request_duration_seconds_count{` + api + `}[{{.window}}]))` +
				` - sum(rate(http_server_request_duration_seconds_bucket{` + api + `,le="0.25"}[{{.window}}]))`,
			totalQuery: `sum(rate(http_server_request_duration_seconds_count{` + api + `}[{{.window}}]))`,
		},
		"consumer-success": {
			service: "orders-consumer", objective: 99.9,
			errorQuery: `sum(rate(messaging_process_duration_seconds_count{` + consumer + `,error_type!=""}[{{.window}}])) or vector(0)`,
			totalQuery: `sum(rate(messaging_process_duration_seconds_count{` + consumer + `}[{{.window}}]))`,
		},
		"consumer-latency": {
			service: "orders-consumer", objective: 99,
			errorQuery: `sum(rate(messaging_process_duration_seconds_count{` + consumer + `}[{{.window}}]))` +
				` - sum(rate(messaging_process_duration_seconds_bucket{` + consumer + `,le="1"}[{{.window}}]))`,
			totalQuery: `sum(rate(messaging_process_duration_seconds_count{` + consumer + `}[{{.window}}]))`,
		},
		"outbox-freshness": {
			service: "orders-worker", objective: 99,
			errorRatioQuery: `avg_over_time((max(orders_outbox_oldest_pending_age_seconds{job="boilerplate/orders-worker"})` +
				` > bool 30)[{{.window}}:1m])`,
		},
	}, got)
}

type sloRule struct {
	Alert       string            `yaml:"alert"`
	Record      string            `yaml:"record"`
	Expr        string            `yaml:"expr"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

func readSLORules(t *testing.T, root, svc string) []sloRule {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, sloRulesDir, svc+".yaml")) //nolint:gosec // Test temporary directory.
	require.NoError(t, err)

	var file struct {
		Groups []struct {
			Rules []sloRule `yaml:"rules"`
		} `yaml:"groups"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &file))

	var rules []sloRule
	for _, group := range file.Groups {
		rules = append(rules, group.Rules...)
	}

	return rules
}

func TestGenerate_RunsSlothIntoPageAndTicketBurnRateAlertsPerSLO(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, []service.Service{ordersService}))

	alerts := map[string]string{}

	for _, rule := range readSLORules(t, root, "orders") {
		if rule.Alert == "" {
			continue
		}

		origin := rule.Labels["service"] + " " + rule.Labels["sloth_slo"]
		alerts[origin] += rule.Labels["severity"] + "/" + rule.Labels["sloth_severity"] + " "

		assert.Equal(t, "SLOBurnRate", rule.Alert, origin)
		assert.Equal(t, []string{"service", "severity", "sloth_severity", "sloth_slo"}, sortedKeys(rule.Labels), origin)
		assert.Equal(t, "https://github.com/lcnascimento/boilerplate/blob/main/docs/runbooks/SLOBurnRate.md",
			rule.Annotations["runbook_url"], origin)
		assert.Equal(t, "http://localhost:3010/d/boilerplate-orders", rule.Annotations["dashboard_url"], origin)
		assert.NotEmpty(t, rule.Annotations["summary"], origin)
		assert.NotEmpty(t, rule.Annotations["description"], origin)
	}

	assert.Equal(t, map[string]string{
		"orders-api api-availability":      "critical/page warning/ticket ",
		"orders-api api-latency":           "critical/page warning/ticket ",
		"orders-consumer consumer-success": "critical/page warning/ticket ",
		"orders-consumer consumer-latency": "critical/page warning/ticket ",
		"orders-worker outbox-freshness":   "critical/page warning/ticket ",
	}, alerts)
	assert.FileExists(t, filepath.Join(runbooksDir, "SLOBurnRate.md"))
}

func TestGenerate_EvaluatesTheSLIOverTheWhole30DayWindowWithoutOptimizing(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, []service.Service{ordersService}))

	periods := map[string]string{}

	for _, rule := range readSLORules(t, root, "orders") {
		if rule.Record == "slo:sli_error:ratio_rate30d" {
			periods[rule.Labels["sloth_slo"]] = rule.Expr
		}
	}

	assert.Equal(t, map[string]string{
		"api-availability": "(sum(rate(http_server_request_total{" +
			`job="boilerplate/orders-api",http_route!~"/ping|/health",http_response_status_code=~"5.."}[30d])) or vector(0))` +
			"\n/\n" +
			`(sum(rate(http_server_request_total{job="boilerplate/orders-api",http_route!~"/ping|/health"}[30d])))` + "\n",
		"api-latency": "(sum(rate(http_server_request_duration_seconds_count{" + `job="boilerplate/orders-api",` +
			`http_route!~"/ping|/health"}[30d])) - sum(rate(http_server_request_duration_seconds_bucket{` +
			`job="boilerplate/orders-api",http_route!~"/ping|/health",le="0.25"}[30d])))` + "\n/\n" +
			"(sum(rate(http_server_request_duration_seconds_count{" +
			`job="boilerplate/orders-api",http_route!~"/ping|/health"}[30d])))` + "\n",
		"consumer-success": `(sum(rate(messaging_process_duration_seconds_count{job="boilerplate/orders-consumer",` +
			`error_type!=""}[30d])) or vector(0))` + "\n/\n" +
			`(sum(rate(messaging_process_duration_seconds_count{job="boilerplate/orders-consumer"}[30d])))` + "\n",
		"consumer-latency": `(sum(rate(messaging_process_duration_seconds_count{job="boilerplate/orders-consumer"}[30d]))` +
			` - sum(rate(messaging_process_duration_seconds_bucket{job="boilerplate/orders-consumer",le="1"}[30d])))` +
			"\n/\n" +
			`(sum(rate(messaging_process_duration_seconds_count{job="boilerplate/orders-consumer"}[30d])))` + "\n",
		"outbox-freshness": `(avg_over_time((max(orders_outbox_oldest_pending_age_seconds{job="boilerplate/orders-worker"})` +
			` > bool 30)[30d:1m]))`,
	}, periods)
}

func TestGenerate_AddsTheExtraSLOAProcessDeclares(t *testing.T) {
	root := t.TempDir()
	createOrder := `{job="boilerplate/orders-api",http_route="/orders/v1/orders"}`
	declared := []service.Service{{
		Name:      "orders",
		Processes: []service.Process{service.Daemon("orders-api", service.API)},
		SLOs: []service.SLO{{
			Process:     "orders-api",
			Name:        "api-create-order-latency",
			Objective:   99.5,
			Description: "Pedidos criados em até 500ms.",
			SLI: service.EventsSLI(
				`sum(rate(http_server_request_duration_seconds_bucket`+createOrder+`[{{.window}}]))`,
				`sum(rate(http_server_request_duration_seconds_count`+createOrder+`[{{.window}}]))`,
			),
		}},
	}}

	err := generate(root, declared)
	require.NoError(t, err, describe(err))

	slos := readSLOSpec(t, root, "orders").SLOs
	names := make([]string, 0, len(slos))

	for _, slo := range slos {
		names = append(names, slo.Name+" "+slo.Labels["service"])
	}

	assert.Equal(t, []string{
		"api-availability orders-api", "api-latency orders-api", "api-create-order-latency orders-api",
	}, names)

	alerts := 0

	for _, rule := range readSLORules(t, root, "orders") {
		if rule.Alert != "" && strings.Contains(rule.Expr, `sloth_slo="api-create-order-latency"`) {
			alerts++
		}
	}

	assert.Equal(t, 2, alerts)
}

func TestGenerate_RejectsAnInvalidExtraSLO(t *testing.T) {
	sli := service.EventsSLI(`sum(rate(target_info[{{.window}}]))`, `sum(rate(target_info[{{.window}}]))`)

	for name, tc := range map[string]struct {
		slo    service.SLO
		reason string
	}{
		"named after an inherited SLO": {
			slo:    service.SLO{Process: "orders-api", Name: "api-latency", Objective: 99, Description: "d", SLI: sli},
			reason: `service "orders" declares SLO "api-latency" twice, in orders-api and orders-api`,
		},
		"on a one-shot process": {
			slo:    service.SLO{Process: "orders-migrate", Name: "api-extra", Objective: 99, Description: "d", SLI: sli},
			reason: `SLO "api-extra" on "orders-migrate", which is not one of its long-running processes`,
		},
		"on a process of another service": {
			slo:    service.SLO{Process: "billing-api", Name: "api-extra", Objective: 99, Description: "d", SLI: sli},
			reason: `SLO "api-extra" on "billing-api", which is not one of its long-running processes`,
		},
		"without the window placeholder": {
			slo: service.SLO{
				Process: "orders-api", Name: "api-extra", Objective: 99, Description: "d",
				SLI: service.RawSLI(`avg_over_time(target_info[5m])`),
			},
			reason: `does not range over {{.window}}`,
		},
		"with an objective of 100%": {
			slo:    service.SLO{Process: "orders-api", Name: "api-extra", Objective: 100, Description: "d", SLI: sli},
			reason: `the objective 100 is not a percentage in (0, 100)`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			declared := []service.Service{{
				Name:      "orders",
				Processes: []service.Process{service.Daemon("orders-api", service.API), service.OneShot("orders-migrate")},
				SLOs:      []service.SLO{tc.slo},
			}}

			err := generate(t.TempDir(), declared)

			require.Error(t, err)
			assert.Contains(t, describe(err), tc.reason)
		})
	}
}
