package logrules

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/grafana/loki/v3/pkg/logql/syntax"
	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

// OpenTelemetry's FATAL range starts at 21, where go-kit's Critical and Fatal levels land.
const severityFatal = 21

const yamlIndent = 2

type ruleFile struct {
	Groups []ruleGroup `yaml:"groups"`
}

type ruleGroup struct {
	Name  string          `yaml:"name"`
	Rules []recordingRule `yaml:"rules"`
}

type recordingRule struct {
	Record string            `yaml:"record"`
	Expr   string            `yaml:"expr"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

func Kind() artifact.Kind {
	return artifact.Kind{
		Dir: "config/loki/rules/fake", Glob: "*.yaml", LoadedBy: artifact.LokiRuler, Build: build, Queries: queries,
	}
}

func build(services []service.Service) ([]artifact.File, error) {
	files := make([]artifact.File, 0, len(services))

	for _, svc := range services {
		group, err := serviceGroup(&svc)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer

		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(yamlIndent)

		if err := encoder.Encode(ruleFile{Groups: []ruleGroup{group}}); err != nil {
			return nil, errors.New("encoding log rules %s", svc.Name).WithCause(err)
		}

		files = append(files, artifact.File{Name: svc.Name + ".yaml", Content: buf.Bytes()})
	}

	return files, nil
}

func streams(names ...string) string {
	if len(names) == 1 {
		return `{service_namespace=` + strconv.Quote(config.Namespace) + `, service_name=` + strconv.Quote(names[0]) + `}`
	}

	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, regexp.QuoteMeta(name))
	}

	return `{service_namespace=` + strconv.Quote(config.Namespace) + `, service_name=~` + strconv.Quote(strings.Join(quoted, "|")) + `}`
}

func recording(record, expr string, labels map[string]string) (recordingRule, error) {
	parsed, err := syntax.ParseSampleExpr(expr)
	if err != nil {
		return recordingRule{}, errors.New("parsing %s %q", record, expr).WithCause(err)
	}

	return recordingRule{Record: record, Expr: parsed.String(), Labels: labels}, nil
}

func serviceGroup(svc *service.Service) (ruleGroup, error) {
	processes := make([]string, 0, len(svc.Processes))
	for _, p := range svc.Processes {
		processes = append(processes, p.Name())
	}

	all := streams(processes...)

	records, err := recording(catalog.ServiceLogRecords,
		"sum by (service, level, "+config.EnvironmentLabel+") (rate("+all+" | label_format service=service_name, level=detected_level [1m]))", nil)
	if err != nil {
		return ruleGroup{}, errors.New("log rules of %s", svc.Name).WithCause(err)
	}

	critical, err := recording(catalog.ServiceLogRecordsCritical,
		"sum by (service, "+config.EnvironmentLabel+") (rate("+all+" | severity_number >= "+strconv.Itoa(severityFatal)+
			" | label_format service=service_name [1m]))", nil)
	if err != nil {
		return ruleGroup{}, errors.New("log rules of %s", svc.Name).WithCause(err)
	}

	rules := []recordingRule{records, critical}

	for _, process := range svc.Processes {
		for _, alert := range process.LogAlerts() {
			rule, err := recording(catalog.ServiceLogAlertRecords,
				"sum by (service, "+config.EnvironmentLabel+") (rate("+streams(process.Name())+" "+alert.Pipeline+" | label_format service=service_name [1m]))",
				map[string]string{"log_alert": alert.Name})
			if err != nil {
				return ruleGroup{}, errors.New("log alert %s of %s", alert.Name, process.Name()).WithCause(err)
			}

			rules = append(rules, rule)
		}
	}

	return ruleGroup{Name: svc.Name, Rules: rules}, nil
}

func queries(content []byte) ([]artifact.Query, error) {
	var file ruleFile
	if err := yaml.Unmarshal(content, &file); err != nil {
		return nil, errors.New("parsing log rules").WithCause(err)
	}

	var out []artifact.Query

	for _, group := range file.Groups {
		for _, rule := range group.Rules {
			out = append(out, artifact.Query{
				Language: artifact.LogQL,
				Expr:     rule.Expr,
				Origin:   "group " + group.Name + " rule " + rule.Record + " " + rule.Labels["log_alert"],
			})
		}
	}

	return out, nil
}
