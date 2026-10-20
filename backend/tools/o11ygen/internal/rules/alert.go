package rules

import (
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
)

type severity string

const (
	critical severity = "critical"
	warning  severity = "warning"
)

type alert struct {
	name        string
	expr        string
	forDuration time.Duration
	severity    severity
	summary     string
	description string
	dashboard   string
	service     string
}

func (a *alert) rule() (rulefmt.Rule, error) {
	switch {
	case a.name == "" || a.expr == "" || a.service == "":
		return rulefmt.Rule{}, errors.New("alert %q has no name, expression or service", a.name)
	case a.severity != critical && a.severity != warning:
		return rulefmt.Rule{}, errors.New("alert %s has severity %q, not critical or warning", a.name, a.severity)
	case a.summary == "" || a.description == "":
		return rulefmt.Rule{}, errors.New("alert %s has no summary or description", a.name)
	case a.dashboard == "":
		return rulefmt.Rule{}, errors.New("alert %s points to no dashboard", a.name)
	}

	return rulefmt.Rule{
		Alert:  a.name,
		Expr:   a.expr,
		For:    model.Duration(a.forDuration),
		Labels: map[string]string{"service": a.service, "severity": string(a.severity)},
		Annotations: map[string]string{
			"summary":       a.summary,
			"description":   a.description,
			"runbook_url":   config.RunbookBaseURL + "/" + a.name + ".md",
			"dashboard_url": config.GrafanaURL + "/d/" + config.Namespace + "-" + a.dashboard,
		},
	}, nil
}

type recording struct {
	name string
	expr string
}

func (r *recording) rule() (rulefmt.Rule, error) {
	if r.name == "" || r.expr == "" {
		return rulefmt.Rule{}, errors.New("recording rule %q has no name or expression", r.name)
	}

	return rulefmt.Rule{Record: r.name, Expr: r.expr}, nil
}
