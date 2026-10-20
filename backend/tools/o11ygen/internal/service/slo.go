package service

import (
	"regexp"
	"slices"
	"strings"

	"github.com/lcnascimento/go-kit/errors"
)

// SLOWindow is the placeholder every SLI query uses for the window Sloth evaluates it over.
const SLOWindow = "{{.window}}"

var sloName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)+$`)

// SLO is an objective over a 30-day window that Process emits, beyond the ones it inherits from its components. Name
// follows <component>-<sli> and is unique within the service.
type SLO struct {
	Process     string
	Name        string
	Objective   float64
	Description string
	SLI         SLI
}

// SLI is built only by EventsSLI or RawSLI.
type SLI struct {
	errorQuery      string
	totalQuery      string
	errorRatioQuery string
}

// EventsSLI divides the bad events by every event; both queries must range over SLOWindow.
func EventsSLI(errorQuery, totalQuery string) SLI {
	return SLI{errorQuery: errorQuery, totalQuery: totalQuery}
}

// RawSLI is a query that already yields the error ratio (0-1) over SLOWindow.
func RawSLI(errorRatioQuery string) SLI {
	return SLI{errorRatioQuery: errorRatioQuery}
}

func (s SLI) Events() (errorQuery, totalQuery string, ok bool) {
	return s.errorQuery, s.totalQuery, s.errorQuery != ""
}

func (s SLI) Raw() (errorRatioQuery string, ok bool) {
	return s.errorRatioQuery, s.errorRatioQuery != ""
}

func validateSLOs(svc Service) error {
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

	queries := []string{s.SLI.errorRatioQuery}
	if errorQuery, totalQuery, ok := s.SLI.Events(); ok {
		queries = []string{errorQuery, totalQuery}
	}

	for _, query := range queries {
		if !strings.Contains(query, SLOWindow) {
			return errors.New("the SLI query %q does not range over %s", query, SLOWindow)
		}
	}

	return nil
}
