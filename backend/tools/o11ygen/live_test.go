package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

// The backend directory: check-live reads the generated files the stack mounts, not a fresh generation.
const backendRoot = "../.."

const (
	liveTimeout  = 2 * time.Minute
	livePoll     = 5 * time.Second
	queryTimeout = 10 * time.Second
)

type liveStack struct {
	prometheus string
	loki       string
	grafana    string
	generated  generatedArtifacts
}

func TestLive(t *testing.T) {
	t.Parallel()

	stack := &liveStack{prometheus: os.Getenv("O11Y_LIVE_PROMETHEUS_URL")}
	if stack.prometheus == "" {
		t.Skip("O11Y_LIVE_PROMETHEUS_URL is not set: run make o11y.check-live against a running stack")
	}

	stack.loki = requireEnv(t, "O11Y_LIVE_LOKI_URL")
	stack.grafana = requireEnv(t, "O11Y_LIVE_GRAFANA_URL")
	stack.generated = readGenerated(t)

	for name, check := range map[string]func(t *testing.T){
		"PresenceAlwaysMetricsExist":        stack.checkPresence,
		"SeriesStayUnderEveryServiceBudget": stack.checkSeriesBudgets,
		"PrometheusRuleGroupsAreHealthy":    stack.checkPrometheusRules,
		"LokiRuleGroupsAreHealthy":          stack.checkLokiRules,
		"GeneratedDashboardsAreProvisioned": stack.checkDashboards,
		"ProvisionedDatasourcesAreHealthy":  stack.checkDatasources,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			check(t)
		})
	}
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()

	value := os.Getenv(name)
	require.NotEmpty(t, value, "%s must be set along with O11Y_LIVE_PROMETHEUS_URL", name)

	return strings.TrimSuffix(value, "/")
}

func eventually(t *testing.T, check func(c *assert.CollectT)) {
	t.Helper()

	assert.EventuallyWithT(t, check, liveTimeout, livePoll)
}

func (s *liveStack) checkPresence(t *testing.T) {
	t.Helper()

	probes, err := catalog.LiveProbes(services)
	require.NoError(t, err)

	eventually(t, func(c *assert.CollectT) {
		for _, probe := range probes {
			result, err := s.query(probe.Expr)
			if assert.NoError(c, err, probe.Origin) {
				assert.NotEmpty(c, result, "%s is absent: %s", probe.Origin, probe.Expr)
			}
		}
	})
}

func (s *liveStack) checkSeriesBudgets(t *testing.T) {
	t.Helper()

	eventually(t, func(c *assert.CollectT) {
		for i := range services {
			svc := &services[i]

			for _, process := range svc.Processes {
				if !process.LongRunning() {
					continue
				}

				job := config.Namespace + "/" + process.Name()
				expr := fmt.Sprintf("%s{job=%q}", catalog.JobSeriesCount, job)

				result, err := s.query(expr)
				if !assert.NoError(c, err, expr) || !assert.Len(c, result, 1, "%s has no series count", job) {
					continue
				}

				count, err := result[0].number()
				if assert.NoError(c, err, expr) {
					assert.Less(c, count, float64(svc.EffectiveSeriesBudget()), "%s is over its series budget", job)
				}
			}
		}
	})
}

func (s *liveStack) checkPrometheusRules(t *testing.T) {
	t.Helper()

	eventually(t, func(c *assert.CollectT) {
		checkRuleGroups(c, s.prometheus+"/api/v1/rules", s.generated.prometheusGroups)
	})
}

func (s *liveStack) checkLokiRules(t *testing.T) {
	t.Helper()

	eventually(t, func(c *assert.CollectT) {
		checkRuleGroups(c, s.loki+"/prometheus/api/v1/rules", s.generated.lokiGroups)
	})
}

type grafanaObject struct {
	UID string `json:"uid"`
}

func (s *liveStack) checkDashboards(t *testing.T) {
	t.Helper()

	require.NotEmpty(t, s.generated.dashboardUIDs)

	eventually(t, func(c *assert.CollectT) {
		var found []grafanaObject
		if !assert.NoError(c, getJSON(s.grafana+"/api/search?type=dash-db&limit=5000", &found)) {
			return
		}

		provisioned := map[string]bool{}
		for _, board := range found {
			provisioned[board.UID] = true
		}

		for _, uid := range s.generated.dashboardUIDs {
			assert.True(c, provisioned[uid], "dashboard %s is not provisioned in Grafana", uid)
		}
	})
}

func (s *liveStack) checkDatasources(t *testing.T) {
	t.Helper()

	provisioned := provisionedDatasources(t)

	eventually(t, func(c *assert.CollectT) {
		var found []grafanaObject
		if !assert.NoError(c, getJSON(s.grafana+"/api/datasources", &found)) {
			return
		}

		uids := make([]string, 0, len(found))
		for _, datasource := range found {
			uids = append(uids, datasource.UID)
		}

		for _, uid := range provisioned {
			assert.Contains(c, uids, uid, "datasource %s is not provisioned in Grafana", uid)
		}

		for _, uid := range uids {
			var health struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			}

			err := getJSON(s.grafana+"/api/datasources/uid/"+url.PathEscape(uid)+"/health", &health)
			if assert.NoError(c, err, "datasource %s", uid) {
				assert.Equal(c, "OK", health.Status, "datasource %s: %s", uid, health.Message)
			}
		}
	})
}

type ruleGroup struct {
	file  string
	name  string
	rules []string
}

type generatedArtifacts struct {
	prometheusGroups []ruleGroup
	lokiGroups       []ruleGroup
	dashboardUIDs    []string
}

func readGenerated(t *testing.T) generatedArtifacts {
	t.Helper()

	var generated generatedArtifacts

	for _, kind := range kinds() {
		paths, err := filepath.Glob(filepath.Join(backendRoot, kind.Dir, kind.Glob))
		require.NoError(t, err)

		for _, file := range paths {
			content, err := os.ReadFile(file) //nolint:gosec // The path comes from globbing a generated kind's directory.
			require.NoError(t, err)

			// The stack reports a rule file by its path inside the container, which ends with the kind's directory.
			name := path.Join(filepath.Base(kind.Dir), filepath.Base(file))

			switch kind.LoadedBy {
			case artifact.Prometheus:
				generated.prometheusGroups = append(generated.prometheusGroups, ruleGroups(t, name, content)...)
			case artifact.LokiRuler:
				generated.lokiGroups = append(generated.lokiGroups, ruleGroups(t, name, content)...)
			case artifact.Grafana:
				var board grafanaObject

				require.NoError(t, json.Unmarshal(content, &board), file)
				require.NotEmpty(t, board.UID, "%s has no uid", file)

				generated.dashboardUIDs = append(generated.dashboardUIDs, board.UID)
			case artifact.Sloth, artifact.Promtool:
			default:
				t.Fatalf("kind %s/%s has no loader check-live knows how to verify", kind.Dir, kind.Glob)
			}
		}
	}

	return generated
}

func ruleGroups(t *testing.T, file string, content []byte) []ruleGroup {
	t.Helper()

	var parsed struct {
		Groups []struct {
			Name  string `yaml:"name"`
			Rules []struct {
				Alert  string `yaml:"alert"`
				Record string `yaml:"record"`
			} `yaml:"rules"`
		} `yaml:"groups"`
	}

	require.NoError(t, yaml.Unmarshal(content, &parsed), file)

	groups := make([]ruleGroup, 0, len(parsed.Groups))

	for _, group := range parsed.Groups {
		rules := make([]string, 0, len(group.Rules))
		for _, rule := range group.Rules {
			rules = append(rules, rule.Alert+rule.Record)
		}

		slices.Sort(rules)
		groups = append(groups, ruleGroup{file: file, name: group.Name, rules: rules})
	}

	return groups
}

func provisionedDatasources(t *testing.T) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(backendRoot, "config", "grafana", "provisioning", "datasources", "*.yaml"))
	require.NoError(t, err)

	uids := make([]string, 0, len(paths))

	for _, file := range paths {
		content, err := os.ReadFile(file) //nolint:gosec // The path comes from globbing the provisioning directory.
		require.NoError(t, err)

		var provisioning struct {
			Datasources []grafanaObject `yaml:"datasources"`
		}

		require.NoError(t, yaml.Unmarshal(content, &provisioning), file)

		for _, datasource := range provisioning.Datasources {
			uids = append(uids, datasource.UID)
		}
	}

	require.NotEmpty(t, uids, "no datasource is provisioned")

	return uids
}

type loadedGroup struct {
	Name  string       `json:"name"`
	File  string       `json:"file"`
	Rules []loadedRule `json:"rules"`
}

type loadedRule struct {
	Name      string `json:"name"`
	Health    string `json:"health"`
	LastError string `json:"lastError"` //nolint:tagliatelle // The Prometheus rules API names it so.
}

func checkRuleGroups(c *assert.CollectT, endpoint string, want []ruleGroup) {
	var response struct {
		Data struct {
			Groups []loadedGroup `json:"groups"`
		} `json:"data"`
	}

	if !assert.NoError(c, getJSON(endpoint, &response)) {
		return
	}

	for _, group := range want {
		index := slices.IndexFunc(response.Data.Groups, func(loaded loadedGroup) bool {
			return loaded.Name == group.name && (loaded.File == group.file || strings.HasSuffix(loaded.File, "/"+group.file))
		})
		if !assert.GreaterOrEqual(c, index, 0, "group %s of %s is not loaded by %s", group.name, group.file, endpoint) {
			continue
		}

		loaded := response.Data.Groups[index]
		names := make([]string, 0, len(loaded.Rules))

		for _, rule := range loaded.Rules {
			names = append(names, rule.Name)
			assert.Equal(c, "ok", rule.Health, "rule %s of group %s: %s", rule.Name, group.name, rule.LastError)
		}

		slices.Sort(names)
		assert.Equal(c, group.rules, names, "rules of group %s of %s", group.name, group.file)
	}
}

type sample struct {
	Value [2]any `json:"value"`
}

func (s sample) number() (float64, error) {
	text, ok := s.Value[1].(string)
	if !ok {
		return 0, errors.New("sample value %v is not a string", s.Value[1])
	}

	return strconv.ParseFloat(text, 64)
}

func (s *liveStack) query(expr string) ([]sample, error) {
	var response struct {
		Data struct {
			Result []sample `json:"result"`
		} `json:"data"`
	}

	err := getJSON(s.prometheus+"/api/v1/query?query="+url.QueryEscape(expr), &response)

	return response.Data.Result, err
}

// getJSON sends the URL's user info, if any, as basic auth (Grafana's admin).
func getJSON(endpoint string, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New("GET %s: %s", request.URL.Redacted(), response.Status)
	}

	return json.NewDecoder(response.Body).Decode(out)
}
