package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const lokiRulesDir = "config/loki/rules/fake"

type lokiRuleFile struct {
	Groups []struct {
		Name  string `yaml:"name"`
		Rules []struct {
			Record string            `yaml:"record"`
			Expr   string            `yaml:"expr"`
			Labels map[string]string `yaml:"labels"`
		} `yaml:"rules"`
	} `yaml:"groups"`
}

func readLokiRules(t *testing.T, root, name string) map[string]map[string]string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, lokiRulesDir, name)) //nolint:gosec // The path comes from the test's temporary directory.
	require.NoError(t, err)

	var file lokiRuleFile
	require.NoError(t, yaml.Unmarshal(raw, &file))

	out := map[string]map[string]string{}

	for _, group := range file.Groups {
		out[group.Name] = map[string]string{}

		for _, rule := range group.Rules {
			key := rule.Record
			if alert := rule.Labels["log_alert"]; alert != "" {
				key += ` log_alert=` + alert
			}

			out[group.Name][key] = rule.Expr
		}
	}

	return out
}

func lokiRuleFiles(t *testing.T, root string) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(root, lokiRulesDir, "*"))
	require.NoError(t, err)

	names := make([]string, 0, len(paths))
	for _, path := range paths {
		names = append(names, filepath.Base(path))
	}

	return names
}

func TestGenerate_RecordsTheLogMetricsOfEveryProcessOfAServiceInTheFakeTenant(t *testing.T) {
	root := t.TempDir()
	declared := []service.Service{{
		Name: "orders",
		Processes: []service.Process{
			service.Daemon("orders-api", service.API),
			service.Daemon("orders-worker", service.Worker),
			service.OneShot("orders-migrate"),
		},
	}}

	require.NoError(t, generate(root, declared))

	assert.Equal(t, []string{"orders.yaml"}, lokiRuleFiles(t, root))
	assert.Equal(t, map[string]map[string]string{
		"orders": {
			"service:log_records:rate1m": `sum by (service,level)(rate({service_namespace="boilerplate", ` +
				`service_name=~"orders-api|orders-worker|orders-migrate"} ` +
				`| label_format service=service_name,level=detected_level[1m]))`,
			"service:log_records_critical:rate1m": `sum by (service)(rate({service_namespace="boilerplate", ` +
				`service_name=~"orders-api|orders-worker|orders-migrate"} ` +
				`| severity_number>=21 | label_format service=service_name[1m]))`,
		},
	}, readLokiRules(t, root, "orders.yaml"))
}

func TestGenerate_WritesNoLogRulesWithoutServices(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, lokiRulesDir)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "removed-service.yaml"), []byte("groups: []\n"), 0o600))

	require.NoError(t, generate(root, nil))

	assert.Empty(t, lokiRuleFiles(t, root))
}

func TestGenerate_AlertsOnACriticalLogOfEveryLongRunningProcess(t *testing.T) {
	root := t.TempDir()
	declared := []service.Service{{
		Name: "orders",
		Processes: []service.Process{
			service.Daemon("orders-api", service.API),
			service.OneShot("orders-migrate"),
		},
	}}

	require.NoError(t, generate(root, declared))

	var critical []string

	for _, group := range readRules(t, filepath.Join(root, rulesDir, "orders.yaml")).Groups {
		for _, rule := range group.Rules {
			if rule.Alert == "CriticalLogRecorded" {
				critical = append(critical, rule.Labels["service"]+" "+rule.Labels["severity"]+" "+rule.Expr)
			}
		}
	}

	assert.Equal(t, []string{
		`orders-api critical sum_over_time(service:log_records_critical:rate1m{service="orders-api"}[5m]) * 60 > 0`,
	}, critical)
}

var paymentDeclined = service.LogAlert{
	Name:            "PaymentDeclinedSpike",
	Process:         "orders-api",
	Pipeline:        `|= "payment declined" | detected_level="warn"`,
	MaxRecordsPer5m: 10,
	Severity:        service.Warning,
	Summary:         "{{ $value | humanize }} pagamentos recusados em 5 minutos",
	Description:     "O gateway recusou mais de 10 pagamentos em 5 minutos. Veja primeiro o status do gateway.",
}

func ordersWith(alerts ...service.LogAlert) []service.Service {
	return []service.Service{{
		Name: "orders",
		Processes: []service.Process{
			service.Daemon("orders-api", service.API),
			service.Daemon("orders-worker", service.Worker),
			service.OneShot("orders-migrate"),
		},
		LogAlerts: alerts,
	}}
}

func TestGenerate_RecordsAndAlertsOnTheLogAlertsAServiceDeclares(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, ordersWith(paymentDeclined)))

	recorded := readLokiRules(t, root, "orders.yaml")["orders"]["service:log_alert_records:rate1m log_alert=PaymentDeclinedSpike"]
	assert.Equal(t, `sum by (service)(rate({service_namespace="boilerplate", service_name="orders-api"} `+
		`|= "payment declined" | detected_level="warn" | label_format service=service_name[1m]))`, recorded)
	assert.NoError(t, catalog.Check(artifact.Query{Language: artifact.LogQL, Expr: recorded}))

	type alert struct {
		Group       string
		Expr        string
		Labels      map[string]string
		Annotations map[string]string
	}

	var declaredAlerts []alert

	for _, group := range readRules(t, filepath.Join(root, rulesDir, "orders.yaml")).Groups {
		for _, rule := range group.Rules {
			if rule.Alert == paymentDeclined.Name {
				declaredAlerts = append(declaredAlerts, alert{Group: group.Name, Expr: rule.Expr, Labels: rule.Labels, Annotations: rule.Annotations})
			}
		}
	}

	require.Len(t, declaredAlerts, 1)
	assert.NoError(t, catalog.Check(artifact.Query{Language: artifact.PromQL, Expr: declaredAlerts[0].Expr}))
	assert.Equal(t, []alert{{
		Group:  "orders-api",
		Expr:   `sum_over_time(service:log_alert_records:rate1m{service="orders-api", log_alert="PaymentDeclinedSpike"}[5m]) * 60 > 10`,
		Labels: map[string]string{"service": "orders-api", "severity": "warning"},
		Annotations: map[string]string{
			"summary":       paymentDeclined.Summary,
			"description":   paymentDeclined.Description,
			"runbook_url":   "https://github.com/lcnascimento/boilerplate/blob/main/docs/runbooks/PaymentDeclinedSpike.md",
			"dashboard_url": "http://localhost:3010/d/boilerplate-orders",
		},
	}}, declaredAlerts)
}

func TestGenerate_RejectsAnInvalidLogAlert(t *testing.T) {
	with := func(change func(*service.LogAlert)) service.LogAlert {
		alert := paymentDeclined
		change(&alert)

		return alert
	}

	cases := map[string]struct {
		alerts []service.LogAlert
		err    string
	}{
		"on a one-shot process": {
			alerts: []service.LogAlert{with(func(a *service.LogAlert) { a.Process = "orders-migrate" })},
			err:    `log alert PaymentDeclinedSpike names "orders-migrate", not a long-running process of service "orders"`,
		},
		"on a process of another service": {
			alerts: []service.LogAlert{with(func(a *service.LogAlert) { a.Process = "billing-api" })},
			err:    `log alert PaymentDeclinedSpike names "billing-api", not a long-running process of service "orders"`,
		},
		"declared twice": {
			alerts: []service.LogAlert{paymentDeclined, paymentDeclined},
			err:    `service "orders" declares log alert PaymentDeclinedSpike twice`,
		},
		"without a pipeline": {
			alerts: []service.LogAlert{with(func(a *service.LogAlert) { a.Pipeline = "" })},
			err:    `service "orders" has a log alert with no name or pipeline`,
		},
		"with a negative threshold": {
			alerts: []service.LogAlert{with(func(a *service.LogAlert) { a.MaxRecordsPer5m = -1 })},
			err:    `log alert PaymentDeclinedSpike has a negative threshold`,
		},
		"with a pipeline that is not LogQL": {
			alerts: []service.LogAlert{with(func(a *service.LogAlert) { a.Pipeline = `|= payment` })},
			err:    `log alert PaymentDeclinedSpike of orders-api`,
		},
		"with an unknown severity": {
			alerts: []service.LogAlert{with(func(a *service.LogAlert) { a.Severity = "page" })},
			err:    `alert PaymentDeclinedSpike has severity "page", not critical or warning`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := generate(t.TempDir(), ordersWith(tc.alerts...))

			require.Error(t, err)
			assert.Contains(t, describe(err), tc.err)
		})
	}
}
