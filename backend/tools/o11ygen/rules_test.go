package main

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	rulesDir      = "config/prometheus/rules"
	ruleFixtures  = "config/prometheus/tests/fixture*.yaml"
	ruleTestsGlob = "../../config/prometheus/tests/*.test.yaml"
	runbooksDir   = "../../../docs/runbooks"
)

type ruleFile struct {
	Groups []struct {
		Name  string `yaml:"name"`
		Rules []struct {
			Alert       string            `yaml:"alert"`
			Record      string            `yaml:"record"`
			Expr        string            `yaml:"expr"`
			Labels      map[string]string `yaml:"labels"`
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"rules"`
	} `yaml:"groups"`
}

func readRules(t *testing.T, file string) ruleFile {
	t.Helper()

	raw := readFile(t, file)

	var rules ruleFile
	require.NoError(t, yaml.Unmarshal(raw, &rules))

	return rules
}

func generatedRules(t *testing.T, root string) map[string]ruleFile {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(root, rulesDir, "*.yaml"))
	require.NoError(t, err)

	files := map[string]ruleFile{}
	for _, path := range paths {
		files[filepath.Base(path)] = readRules(t, path)
	}

	return files
}

func alertsByGroupAndService(rules ruleFile) map[string]map[string][]string {
	out := map[string]map[string][]string{}

	for _, group := range rules.Groups {
		out[group.Name] = map[string][]string{}

		for _, rule := range group.Rules {
			svc := rule.Labels["service"]
			out[group.Name][svc] = append(out[group.Name][svc], rule.Alert+rule.Record)
		}
	}

	return out
}

func TestGenerate_InheritsTheAlertsOfEachProcessFromItsComponents(t *testing.T) {
	root := t.TempDir()
	declared := []service.Service{{
		Name: "orders",
		Processes: []service.Process{
			service.Daemon("orders-api", service.API).WithDBPool(),
			service.Daemon("orders-consumer", service.Consumer).WithConsumerGroup("orders"),
			service.Daemon("orders-worker", service.Worker).WithDBPool(),
			service.OneShot("orders-migrate"),
		},
	}}

	require.NoError(t, generate(root, declared))

	assert.Equal(t, map[string]ruleFile{"infra.yaml": {}, "orders.yaml": {}}, mapKeysOnly(generatedRules(t, root)))
	assert.Equal(t, map[string]map[string][]string{
		"orders-api": {
			"orders-api": {"ServiceTelemetryAbsent", "HealthCheckFailing", "DBPoolSaturated", "SeriesBudgetExceeded", "CriticalLogRecorded"},
		},
		"orders-consumer": {
			"orders-consumer": {
				"ServiceTelemetryAbsent", "HealthCheckFailing", "ConsumerLagHigh", "DLQMessageReceived", "SeriesBudgetExceeded",
				"CriticalLogRecorded",
			},
		},
		"orders-worker": {
			"orders-worker": {
				"ServiceTelemetryAbsent", "HealthCheckFailing", "DBPoolSaturated", "OutboxAgeAbsent", "SeriesBudgetExceeded",
				"CriticalLogRecorded",
			},
		},
	}, alertsByGroupAndService(readRules(t, filepath.Join(root, rulesDir, "orders.yaml"))))
	assert.Equal(t, map[string]map[string][]string{
		"cardinality": {"": {"job:series:count"}},
		"kafka":       {"kafka": {"KafkaBrokersDown"}},
		"postgres":    {"postgres": {"PostgresDown", "PostgresConnectionsNearMax"}},
	}, alertsByGroupAndService(readRules(t, filepath.Join(root, rulesDir, "infra.yaml"))))
}

func mapKeysOnly(files map[string]ruleFile) map[string]ruleFile {
	out := map[string]ruleFile{}
	for name := range files {
		out[name] = ruleFile{}
	}

	return out
}

func TestGenerate_ScopesTheProcessAlertsToTheProcess(t *testing.T) {
	root := t.TempDir()
	declared := []service.Service{{
		Name: "orders",
		Processes: []service.Process{
			service.Daemon("orders-consumer", service.Consumer).WithConsumerGroup("orders.v2").WithDBPool(),
			service.Daemon("orders-worker", service.Worker),
		},
	}}

	require.NoError(t, generate(root, declared))

	exprs := map[string]string{}

	for _, group := range readRules(t, filepath.Join(root, rulesDir, "orders.yaml")).Groups {
		for _, rule := range group.Rules {
			exprs[rule.Labels["service"]+" "+rule.Alert] = rule.Expr
		}
	}

	assert.Equal(t, map[string]string{
		"orders-consumer ServiceTelemetryAbsent": `absent_over_time(target_info{job="boilerplate/orders-consumer"}[5m])`,
		"orders-consumer HealthCheckFailing": `sum by (job) (increase(http_server_request_total{job="boilerplate/orders-consumer",` +
			` http_route="/health", http_response_status_code=~"5.."}[2m])) > 0 unless sum by (job)` +
			` (increase(http_server_request_total{job="boilerplate/orders-consumer", http_route="/health",` +
			` http_response_status_code=~"2.."}[2m])) > 0`,
		"orders-consumer DBPoolSaturated": `max(pgxpool_acquired_connections{job="boilerplate/orders-consumer"}` +
			` / pgxpool_max_connections{job="boilerplate/orders-consumer"}) > 0.9`,
		"orders-consumer ConsumerLagHigh": `sum by (consumergroup)` +
			` (kafka_consumergroup_uncommitted_offsets{consumergroup="orders.v2"}) > 100`,
		"orders-consumer DLQMessageReceived": `sum by (topic)` +
			` (increase(kafka_topic_partition_current_offset{topic=~".+\\.orders\\.v2\\.dlq"}[5m])) > 0`,
		"orders-worker ServiceTelemetryAbsent": `absent_over_time(target_info{job="boilerplate/orders-worker"}[5m])`,
		"orders-worker HealthCheckFailing": `sum by (job) (increase(http_server_request_total{job="boilerplate/orders-worker",` +
			` http_route="/health", http_response_status_code=~"5.."}[2m])) > 0 unless sum by (job)` +
			` (increase(http_server_request_total{job="boilerplate/orders-worker", http_route="/health",` +
			` http_response_status_code=~"2.."}[2m])) > 0`,
		"orders-worker OutboxAgeAbsent": `absent(orders_outbox_oldest_pending_age_seconds` +
			`{job="boilerplate/orders-worker"})`,
		"orders-consumer SeriesBudgetExceeded": `job:series:count{job="boilerplate/orders-consumer"} > 2000`,
		"orders-worker SeriesBudgetExceeded":   `job:series:count{job="boilerplate/orders-worker"} > 2000`,
		"orders-consumer CriticalLogRecorded":  `sum_over_time(service:log_records_critical:rate1m{service="orders-consumer"}[5m]) * 60 > 0`,
		"orders-worker CriticalLogRecorded":    `sum_over_time(service:log_records_critical:rate1m{service="orders-worker"}[5m]) * 60 > 0`,
	}, exprs)
}

func TestGenerate_NamesTheDependenciesEachHealthCheckChecks(t *testing.T) {
	root := t.TempDir()
	declared := []service.Service{
		{
			Name: "orders",
			Processes: []service.Process{
				service.Daemon("orders-api", service.API).WithDBPool(),
				service.Daemon("orders-consumer", service.Consumer).WithConsumerGroup("orders").WithDBPool(),
				service.Daemon("orders-worker", service.Worker),
			},
		},
		{Name: "billing", Processes: []service.Process{service.Daemon("billing-api", service.API)}},
	}

	require.NoError(t, generate(root, declared))

	descriptions := map[string]string{}

	for _, file := range []string{"orders.yaml", "billing.yaml"} {
		for _, group := range readRules(t, filepath.Join(root, rulesDir, file)).Groups {
			for _, rule := range group.Rules {
				if rule.Alert == "HealthCheckFailing" {
					descriptions[rule.Labels["service"]] = rule.Annotations["description"]
				}
			}
		}
	}

	assert.Equal(t, map[string]string{
		"orders-api": "Todo /health de orders-api respondeu 5xx nos últimos 2 minutos: uma dependência que ele checa " +
			"(o Postgres) está fora, e quem faz o probe o considera fora do ar. Veja primeiro nos logs de orders-api " +
			"por que o health check falha e os alertas das dependências (PostgresDown).",
		"orders-consumer": "Todo /health de orders-consumer respondeu 5xx nos últimos 2 minutos: uma dependência que ele " +
			"checa (o Postgres, o consumer e o Kafka) está fora, e quem faz o probe o considera fora do ar. Veja primeiro " +
			"nos logs de orders-consumer por que o health check falha e os alertas das dependências " +
			"(PostgresDown, KafkaBrokersDown).",
		"orders-worker": "Todo /health de orders-worker respondeu 5xx nos últimos 2 minutos: uma dependência que ele " +
			"checa (o forwarder e o Kafka) está fora, e quem faz o probe o considera fora do ar. Veja primeiro nos logs " +
			"de orders-worker por que o health check falha e os alertas das dependências (KafkaBrokersDown).",
	}, descriptions)
}

func TestGenerate_CountsTheSeriesOfEveryJob(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, nil))

	var records []string

	for _, group := range readRules(t, filepath.Join(root, rulesDir, "infra.yaml")).Groups {
		for _, rule := range group.Rules {
			if rule.Record != "" {
				records = append(records, group.Name+" "+rule.Record+" = "+rule.Expr)
			}
		}
	}

	assert.Equal(t, []string{`cardinality job:series:count = count by (job) ({__name__=~".+"})`}, records)
}

func TestGenerate_OverridesTheSeriesBudgetOfAService(t *testing.T) {
	root := t.TempDir()
	declared := []service.Service{
		{Name: "orders", Processes: []service.Process{service.Daemon("orders-api", service.API)}, SeriesBudget: 5000},
		{Name: "billing", Processes: []service.Process{service.Daemon("billing-api", service.API)}},
	}

	require.NoError(t, generate(root, declared))

	budgets := map[string]string{}

	for name, rules := range generatedRules(t, root) {
		for _, group := range rules.Groups {
			for _, rule := range group.Rules {
				if rule.Alert == "SeriesBudgetExceeded" {
					budgets[name+" "+rule.Labels["service"]] = rule.Expr
				}
			}
		}
	}

	assert.Equal(t, map[string]string{
		"orders.yaml orders-api":   `job:series:count{job="boilerplate/orders-api"} > 5000`,
		"billing.yaml billing-api": `job:series:count{job="boilerplate/billing-api"} > 2000`,
	}, budgets)
}

func TestGenerate_RejectsANegativeSeriesBudget(t *testing.T) {
	declared := []service.Service{
		{Name: "orders", Processes: []service.Process{service.Daemon("orders-api", service.API)}, SeriesBudget: -1},
	}

	err := generate(t.TempDir(), declared)

	require.ErrorContains(t, err, `service "orders" declares a negative series budget`)
}

func TestGenerate_WritesOnlyTheInfraRulesWithoutServices(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, nil))

	assert.Equal(t, map[string]ruleFile{"infra.yaml": {}}, mapKeysOnly(generatedRules(t, root)))
}

func TestGenerate_RejectsAServiceNamedAfterTheInfraRules(t *testing.T) {
	declared := []service.Service{{Name: "infra", Processes: []service.Process{service.Daemon("infra-api", service.API)}}}

	err := generate(t.TempDir(), declared)

	require.Error(t, err)
	assert.Contains(t, describe(err), `service name "infra" is reserved`)
}

func TestGenerate_RejectsAConsumerWithoutAConsumerGroup(t *testing.T) {
	declared := []service.Service{{
		Name:      "orders",
		Processes: []service.Process{service.Daemon("orders-consumer", service.Consumer)},
	}}

	err := generate(t.TempDir(), declared)

	require.ErrorContains(t, err, `process "orders-consumer" must declare a consumer group`)
}

func dashboardUIDs(t *testing.T, root string) map[string]bool {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, dashboardsDir))
	require.NoError(t, err)

	uids := map[string]bool{}

	for _, entry := range entries {
		if uid, ok := readDashboard(t, root, entry.Name())["uid"].(string); ok {
			uids[uid] = true
		}
	}

	return uids
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	slices.Sort(out)

	return out
}

func TestGenerate_AlertsCarryTheirLabelsAnnotationsRunbookAndDashboard(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	uids := dashboardUIDs(t, root)
	checked := 0

	for name, rules := range generatedRules(t, root) {
		for _, group := range rules.Groups {
			for _, rule := range group.Rules {
				if rule.Record != "" {
					continue
				}

				checked++
				origin := name + ": " + group.Name + "/" + rule.Alert

				assert.Equal(t, []string{"service", "severity"}, sortedKeys(rule.Labels), origin)
				assert.Contains(t, []string{"critical", "warning"}, rule.Labels["severity"], origin)
				assert.Equal(t, []string{"dashboard_url", "description", "runbook_url", "summary"},
					sortedKeys(rule.Annotations), origin)
				assert.Equal(t, rule.Alert+".md", path.Base(rule.Annotations["runbook_url"]), origin)
				assert.FileExists(t, filepath.Join(runbooksDir, rule.Alert+".md"), origin)

				_, uid, _ := strings.Cut(rule.Annotations["dashboard_url"], "/d/")
				assert.True(t, uids[uid], "%s: dashboard_url %q is not a generated dashboard", origin,
					rule.Annotations["dashboard_url"])
			}
		}
	}

	assert.Positive(t, checked)
}

type ruleTestFile struct {
	Tests []struct {
		AlertRuleTests []struct {
			Alertname string `yaml:"alertname"`
		} `yaml:"alert_rule_test"`
	} `yaml:"tests"`
}

func testedAlerts(t *testing.T) map[string]bool {
	t.Helper()

	paths, err := filepath.Glob(ruleTestsGlob)
	require.NoError(t, err)

	tested := map[string]bool{}

	for _, path := range paths {
		raw := readFile(t, path)

		var file ruleTestFile
		require.NoError(t, yaml.Unmarshal(raw, &file), path)

		for _, test := range file.Tests {
			for _, alert := range test.AlertRuleTests {
				tested[alert.Alertname] = true
			}
		}
	}

	return tested
}

func TestGenerate_EveryAlertHasAHandWrittenRuleTest(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, generate(root, services))

	files := generatedRules(t, root)
	fixtures, err := filepath.Glob(filepath.Join(root, ruleFixtures))
	require.NoError(t, err)
	require.Len(t, fixtures, 2)

	for _, fixture := range fixtures {
		files[filepath.Base(fixture)] = readRules(t, fixture)
	}

	tested := testedAlerts(t)
	checked := 0

	for name, rules := range files {
		for _, group := range rules.Groups {
			for _, rule := range group.Rules {
				if rule.Record != "" {
					continue
				}

				checked++

				assert.True(t, tested[rule.Alert], "%s: alert %s appears in no alert_rule_test of %s", name, rule.Alert,
					ruleTestsGlob)
			}
		}
	}

	assert.Positive(t, checked)
}
