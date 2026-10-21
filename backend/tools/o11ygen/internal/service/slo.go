package service

import (
	"regexp"
	"slices"
	"strings"

	"github.com/lcnascimento/go-kit/errors"
)

const SLOWindow = "{{.window}}"

var sloName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)+$`)

type SLO struct {
	Process     string
	Name        string
	Objective   float64
	Description string
	SLI         SLI
}

type SLI interface {
	queries() []string
}

type EventsSLI struct {
	ErrorQuery string
	TotalQuery string
}

type RawSLI struct {
	ErrorRatioQuery string
}

func (s EventsSLI) queries() []string { return []string{s.ErrorQuery, s.TotalQuery} }

func (s RawSLI) queries() []string { return []string{s.ErrorRatioQuery} }

func validateSLOs(svc *Service) error {
	for i := range svc.SLOs {
		slo := &svc.SLOs[i]

		process := slices.IndexFunc(svc.Processes, func(p Process) bool { return p.name == slo.Process })
		if process < 0 || !svc.Processes[process].LongRunning() {
			return errors.New("service %q declares SLO %q on %q, which is not one of its long-running processes",
				svc.Name, slo.Name, slo.Process)
		}

		if err := slo.validate(); err != nil {
			return errors.New("service %q declares an invalid SLO %q", svc.Name, slo.Name).WithCause(err)
		}
	}

	return nil
}

func (s *SLO) validate() error {
	if !sloName.MatchString(s.Name) {
		return errors.New("the name does not follow <component>-<sli> in lowercase kebab-case")
	}

	if s.Objective <= 0 || s.Objective >= 100 {
		return errors.New("the objective %v is not a percentage in (0, 100)", s.Objective)
	}

	if s.Description == "" {
		return errors.New("the SLO has no description")
	}

	if s.SLI == nil {
		return errors.New("the SLO has no SLI")
	}

	for _, query := range s.SLI.queries() {
		if !strings.Contains(query, SLOWindow) {
			return errors.New("the SLI query %q does not range over %s", query, SLOWindow)
		}
	}

	return nil
}
