package rules

import (
	"bytes"
	"log/slog"
	"slices"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	infraFile  = "infra"
	yamlIndent = 2

	fixtureSeriesBudget = 5
)

var fixture = service.Service{
	Name:         "fixture",
	SeriesBudget: fixtureSeriesBudget,
	Processes: []service.Process{
		service.Daemon("fixture-api", service.API).WithDBPool(),
		service.Daemon("fixture-consumer", service.Consumer).WithConsumerGroup("fixture"),
		service.Daemon("fixture-worker", service.Worker),
		service.OneShot("fixture-migrate"),
	},
}

func Kind() artifact.Kind {
	return artifact.Kind{Dir: "config/prometheus/rules", Glob: "*.yaml", Build: build, Queries: queries}
}

// FixtureKind writes the rules of a service that runs every component, which the hand-written promtool tests load, so
// that they keep testing every inherited alert whatever services a fork declares.
func FixtureKind() artifact.Kind {
	return artifact.Kind{
		Dir:  "config/prometheus/tests",
		Glob: "fixture.yaml",
		Build: func([]service.Service) ([]artifact.File, error) {
			file, err := encode(fixture.Name, append([]group{serviceGroup(&fixture)}, recordingGroups...)...)
			if err != nil {
				return nil, err
			}

			return []artifact.File{file}, nil
		},
		Queries: queries,
	}
}

func AlertNames(dashboard string) []string {
	var names []string

	for _, group := range append([]group{serviceGroup(&fixture)}, infraGroups...) {
		for i := range group.alerts {
			if a := &group.alerts[i]; (a.dashboard == dashboard || slices.Contains(a.listedOn, dashboard)) && !slices.Contains(names, a.name) {
				names = append(names, a.name)
			}
		}
	}

	return names
}

type group struct {
	name    string
	records []recording
	alerts  []alert
}

func serviceGroup(svc *service.Service) group {
	out := group{name: svc.Name}

	for _, proc := range svc.Processes {
		p := process{Process: proc, service: svc}

		for _, inherited := range processAlerts {
			if inherited.inheritedBy(&p) {
				a := inherited.alert(&p)
				a.service = p.Name()
				out.alerts = append(out.alerts, a)
			}
		}
	}

	for i := range svc.LogAlerts {
		out.alerts = append(out.alerts, logAlert(svc.Name, &svc.LogAlerts[i]))
	}

	return out
}

func build(services []service.Service) ([]artifact.File, error) {
	files := make([]artifact.File, 0, len(services)+1)

	for _, svc := range services {
		if svc.Name == infraFile {
			return nil, errors.New("service name %q is reserved for the infrastructure rules", infraFile)
		}

		file, err := encode(svc.Name, serviceGroup(&svc))
		if err != nil {
			return nil, err
		}

		files = append(files, file)
	}

	infra, err := encode(infraFile, append(slices.Clone(infraGroups), recordingGroups...)...)
	if err != nil {
		return nil, err
	}

	return append(files, infra), nil
}

func encode(name string, groups ...group) (artifact.File, error) {
	ruleGroups := rulefmt.RuleGroups{Groups: make([]rulefmt.RuleGroup, 0, len(groups))}

	for _, g := range groups {
		ruleGroup := rulefmt.RuleGroup{Name: g.name, Rules: make([]rulefmt.Rule, 0, len(g.records)+len(g.alerts))}

		for _, r := range g.records {
			rule, err := r.rule()
			if err != nil {
				return artifact.File{}, errors.New("rules %s", name).WithCause(err)
			}

			ruleGroup.Rules = append(ruleGroup.Rules, rule)
		}

		for i := range g.alerts {
			rule, err := g.alerts[i].rule()
			if err != nil {
				return artifact.File{}, errors.New("rules %s", name).WithCause(err)
			}

			ruleGroup.Rules = append(ruleGroup.Rules, rule)
		}

		ruleGroups.Groups = append(ruleGroups.Groups, ruleGroup)
	}

	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(yamlIndent)

	if err := encoder.Encode(ruleGroups); err != nil {
		return artifact.File{}, errors.New("encoding rules %s", name).WithCause(err)
	}

	if _, err := parse(buf.Bytes()); err != nil {
		return artifact.File{}, errors.New("validating rules %s", name).WithCause(err)
	}

	return artifact.File{Name: name + ".yaml", Content: buf.Bytes()}, nil
}

func parse(content []byte) (*rulefmt.RuleGroups, error) {
	groups, errs := rulefmt.Parse(content, false, model.UTF8Validation, parser.NewParser(parser.Options{}),
		slog.New(slog.DiscardHandler))
	if len(errs) > 0 {
		return nil, errors.New("parsing %d invalid rules", len(errs)).WithCause(errs[0])
	}

	return groups, nil
}

func queries(content []byte) ([]artifact.Query, error) {
	groups, err := parse(content)
	if err != nil {
		return nil, err
	}

	var out []artifact.Query

	for _, group := range groups.Groups {
		for _, rule := range group.Rules {
			out = append(out, artifact.Query{
				Language: artifact.PromQL,
				Expr:     rule.Expr,
				Origin:   "group " + group.Name + " rule " + rule.Alert + rule.Record,
			})
		}
	}

	return out, nil
}
