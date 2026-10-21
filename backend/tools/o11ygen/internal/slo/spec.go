package slo

import (
	"bytes"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	alertName  = "SLOBurnRate"
	yamlIndent = 2
)

// The prometheus/v1 spec of Sloth (github.com/slok/sloth/pkg/prometheus/api/v1), limited to the fields generated here.
type spec struct {
	Version    string     `yaml:"version"`
	Service    string     `yaml:"service"`
	SLOPlugins sloPlugins `yaml:"slo_plugins"`
	SLOs       []sloSpec  `yaml:"slos"`
}

type sloPlugins struct {
	Chain []sloPlugin `yaml:"chain"`
}

type sloPlugin struct {
	ID     string         `yaml:"id"`
	Config map[string]any `yaml:"config"`
}

type sloSpec struct {
	Name        string            `yaml:"name"`
	Objective   float64           `yaml:"objective"`
	Description string            `yaml:"description"`
	Labels      map[string]string `yaml:"labels"`
	SLI         sliSpec           `yaml:"sli"`
	Alerting    alerting          `yaml:"alerting"`
}

type sliSpec struct {
	Events *events `yaml:"events,omitempty"`
	Raw    *raw    `yaml:"raw,omitempty"`
}

type events struct {
	ErrorQuery string `yaml:"error_query"`
	TotalQuery string `yaml:"total_query"`
}

type raw struct {
	ErrorRatioQuery string `yaml:"error_ratio_query"`
}

type alerting struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
	PageAlert   alert             `yaml:"page_alert"`
	TicketAlert alert             `yaml:"ticket_alert"`
}

type alert struct {
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

func Declared(svc *service.Service) ([]service.SLO, error) {
	var out []service.SLO

	for _, p := range svc.Processes {
		for _, inherited := range goKitDefaults {
			if svc.Runtime.RunsGoKit() && p.Runs(inherited.component) {
				slo := inherited.slo(p, svc.Name)
				slo.Process = p.Name()
				out = append(out, slo)
			}
		}
	}

	out = append(out, svc.SLOs...)
	seen := map[string]string{}

	for i := range out {
		slo := &out[i]
		if other, ok := seen[slo.Name]; ok {
			return nil, errors.New("service %q declares SLO %q twice, in %s and %s", svc.Name, slo.Name, other, slo.Process)
		}

		seen[slo.Name] = slo.Process
	}

	return out, nil
}

func specOf(svc string, slos []service.SLO) *spec {
	out := &spec{
		Version: "prometheus/v1",
		Service: svc,
		SLOPlugins: sloPlugins{Chain: []sloPlugin{{
			ID:     "sloth.dev/core/sli_rules/v1",
			Config: map[string]any{"disableOptimized": true},
		}}},
	}

	for i := range slos {
		slo := &slos[i]

		var sli sliSpec

		switch declared := slo.SLI.(type) {
		case service.EventsSLI:
			sli.Events = &events{ErrorQuery: declared.ErrorQuery, TotalQuery: declared.TotalQuery}
		case service.RawSLI:
			sli.Raw = &raw{ErrorRatioQuery: declared.ErrorRatioQuery}
		}

		out.SLOs = append(out.SLOs, sloSpec{
			Name:        slo.Name,
			Objective:   slo.Objective,
			Description: slo.Description,
			Labels:      map[string]string{"service": slo.Process},
			SLI:         sli,
			Alerting: alerting{
				Name:   alertName,
				Labels: map[string]string{"service": slo.Process, "sloth_slo": slo.Name},
				Annotations: map[string]string{
					"runbook_url":   config.RunbookBaseURL + "/" + alertName + ".md",
					"dashboard_url": config.GrafanaURL + "/d/" + config.Namespace + "-" + svc,
				},
				PageAlert: alert{
					Labels: map[string]string{"severity": "critical"},
					Annotations: severityAnnotations(slo,
						"queimando o error budget rápido demais", "5m ou 30m",
						"Incidente em andamento: a 14,4x ou 6x o ritmo sustentável, o error budget de 30 dias acaba em 2 a 5 dias. "+
							"Aja agora: veja no dashboard do serviço qual SLI piorou e desde quando, depois os logs de erro de "+
							slo.Process+"."),
				},
				TicketAlert: alert{
					Labels: map[string]string{"severity": "warning"},
					Annotations: severityAnnotations(slo,
						"consumindo o error budget acima do ritmo sustentável", "2h ou 6h",
						"Consumo sustentado de 3x ou 1x o ritmo sustentável: mantido, o error budget de 30 dias se esgota "+
							"antes do fim da janela. Sem urgência: investigue em horário comercial, pelo dashboard do serviço "+
							"(qual SLI piorou e desde quando) e depois pelos logs de erro de "+slo.Process+"."),
				},
			},
		})
	}

	return out
}

// shortWindows names the windows $value comes from: the expr is `(short and long) or (short and long)` and `and`
// keeps the left-hand sample.
func severityAnnotations(slo *service.SLO, state, shortWindows, guidance string) map[string]string {
	title := "SLO " + slo.Name + " de " + slo.Process + " " + state

	return map[string]string{
		"title":       title,
		"summary":     title + ": {{ $value | humanizePercentage }} de eventos ruins na janela curta (" + shortWindows + ")",
		"description": slo.Description + " " + guidance,
	}
}

func encodeSpec(s *spec) ([]byte, error) {
	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(yamlIndent)

	if err := encoder.Encode(s); err != nil {
		return nil, errors.New("encoding the SLO spec of %s", s.Service).WithCause(err)
	}

	return buf.Bytes(), nil
}

type serviceSpec struct {
	name    string
	content []byte
}

func specs(services []service.Service) ([]serviceSpec, error) {
	var out []serviceSpec

	for _, svc := range services {
		slos, err := Declared(&svc)
		if err != nil {
			return nil, err
		}

		if len(slos) == 0 {
			continue
		}

		content, err := encodeSpec(specOf(svc.Name, slos))
		if err != nil {
			return nil, err
		}

		out = append(out, serviceSpec{name: svc.Name, content: content})
	}

	return out, nil
}

func SpecKind() artifact.Kind {
	return artifact.Kind{
		Dir:      "config/sloth",
		LoadedBy: artifact.Sloth,
		Glob:     "*.yaml",
		Build: func(services []service.Service) ([]artifact.File, error) {
			serviceSpecs, err := specs(services)
			if err != nil {
				return nil, err
			}

			files := make([]artifact.File, 0, len(serviceSpecs))
			for _, s := range serviceSpecs {
				files = append(files, artifact.File{Name: s.name + ".yaml", Content: s.content})
			}

			return files, nil
		},
		Queries: specQueries,
	}
}

func specQueries(content []byte) ([]artifact.Query, error) {
	var s spec
	if err := yaml.Unmarshal(content, &s); err != nil {
		return nil, errors.New("parsing an SLO spec").WithCause(err)
	}

	var out []artifact.Query

	add := func(slo, field, query string) {
		out = append(out, artifact.Query{
			Language: artifact.PromQL,
			Expr:     strings.ReplaceAll(query, service.SLOWindow, "5m"),
			Origin:   "service " + s.Service + " slo " + slo + " " + field,
		})
	}

	for i := range s.SLOs {
		slo := &s.SLOs[i]
		if slo.SLI.Events != nil {
			add(slo.Name, "error_query", slo.SLI.Events.ErrorQuery)
			add(slo.Name, "total_query", slo.SLI.Events.TotalQuery)
		}

		if slo.SLI.Raw != nil {
			add(slo.Name, "error_ratio_query", slo.SLI.Raw.ErrorRatioQuery)
		}
	}

	return out, nil
}
