package slo

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	period   = "30d"
	specMode = 0o600
)

// RuleKind runs `sloth generate` on the spec of each service. It owns its own directory: the alert rules kind deletes
// any file it did not write in config/prometheus/rules.
func RuleKind() artifact.Kind {
	return artifact.Kind{
		Dir: "config/prometheus/slo", Glob: "*.yaml", LoadedBy: artifact.Prometheus, Build: buildRules, Queries: ruleQueries,
	}
}

func buildRules(services []service.Service) ([]artifact.File, error) {
	serviceSpecs, err := specs(services)
	if err != nil || len(serviceSpecs) == 0 {
		return nil, err
	}

	sloth, err := exec.LookPath("sloth")
	if err != nil {
		return nil, errors.New("sloth is not in PATH: run make setup, and make o11y.generate, from backend/").WithCause(err)
	}

	dir, err := os.MkdirTemp("", "o11ygen-sloth-")
	if err != nil {
		return nil, errors.New("creating a directory for the SLO specs").WithCause(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()

	files := make([]artifact.File, 0, len(serviceSpecs))

	for _, s := range serviceSpecs {
		content, err := generate(sloth, dir, s)
		if err != nil {
			return nil, err
		}

		files = append(files, artifact.File{Name: s.name + ".yaml", Content: content})
	}

	return files, nil
}

func generate(sloth, dir string, s serviceSpec) ([]byte, error) {
	in := filepath.Join(dir, s.name+".yaml")
	if err := os.WriteFile(in, s.content, specMode); err != nil {
		return nil, errors.New("writing the SLO spec of %s", s.name).WithCause(err)
	}

	out := filepath.Join(dir, s.name+".rules.yaml")

	//nolint:gosec // The binary is the sloth that make setup installs, found through PATH.
	cmd := exec.CommandContext(context.Background(), sloth, "generate", "--no-log", "--default-slo-period", period, "--input", in, "--out", out)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, errors.New("sloth generate %s: %s", s.name, output).WithCause(err)
	}

	content, err := os.ReadFile(out) //nolint:gosec // The path is inside the temporary directory created above.
	if err != nil {
		return nil, errors.New("reading the SLO rules of %s", s.name).WithCause(err)
	}

	return normalize(s.name, content)
}

// normalize re-encodes the rules Sloth writes without folding long lines, so that renaming the namespace (make init)
// leaves the file byte-identical to a regeneration.
func normalize(name string, content []byte) ([]byte, error) {
	groups, err := parse(content)
	if err != nil {
		return nil, errors.New("validating the SLO rules of %s", name).WithCause(err)
	}

	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(yamlIndent)

	if err := encoder.Encode(groups); err != nil {
		return nil, errors.New("encoding the SLO rules of %s", name).WithCause(err)
	}

	return buf.Bytes(), nil
}

func parse(content []byte) (*rulefmt.RuleGroups, error) {
	groups, errs := rulefmt.Parse(content, false, model.UTF8Validation, parser.NewParser(parser.Options{}),
		slog.New(slog.DiscardHandler))
	if len(errs) > 0 {
		return nil, errors.New("parsing %d invalid rules", len(errs)).WithCause(errs[0])
	}

	return groups, nil
}

func ruleQueries(content []byte) ([]artifact.Query, error) {
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
