package catalog

import (
	"fmt"
	"slices"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

type Emitter func(svc *service.Service, process service.Process) bool

var (
	everyProcess Emitter = func(_ *service.Service, process service.Process) bool { return process.LongRunning() }
	dbPool       Emitter = func(_ *service.Service, process service.Process) bool { return process.DBPool() }
)

func runs(components ...service.Component) Emitter {
	return func(_ *service.Service, process service.Process) bool {
		return slices.ContainsFunc(components, process.Runs)
	}
}

func outboxOf(serviceName string) Emitter {
	return func(svc *service.Service, process service.Process) bool {
		return svc.Name == serviceName && process.Runs(service.Worker)
	}
}

// liveWindow spans a whole CI run, so a process that logged or exported once since it started counts as present.
const liveWindow = "15m"

// processSelectors scopes a series to the process that emits it; a source missing here carries no process label.
var processSelectors = map[Source]func(process string) string{
	OTLP:              jobSelector,
	OTLPResource:      jobSelector,
	RecordingRule:     jobSelector,
	LokiRecordingRule: func(process string) string { return fmt.Sprintf(`{service=%q}`, process) },
	SlothRecording:    func(process string) string { return fmt.Sprintf(`{service=%q}`, process) },
}

var prometheusSources = map[Source]bool{
	OTLP: true, OTLPResource: true, Scrape: true, RecordingRule: true, LokiRecordingRule: true, SlothRecording: true,
}

func jobSelector(process string) string {
	return fmt.Sprintf(`{job=%q}`, config.Namespace+"/"+process)
}

// LiveProbes are the queries that must return a series on the running stack: one per PresenceAlways metric the stack
// emits, and one per process that emits it for the metrics the declared services emit.
func LiveProbes(services []service.Service) ([]artifact.Query, error) {
	names := make([]string, 0, len(metrics))
	for name := range metrics {
		names = append(names, name)
	}

	slices.Sort(names)

	var probes []artifact.Query

	for _, name := range names {
		metric := metrics[name]
		if metric.Presence != PresenceAlways {
			continue
		}

		if !prometheusSources[metric.Source] {
			return nil, errors.New("metric %s comes from a source check-live does not know how to probe", name)
		}

		probes = append(probes, metric.probes(name, services)...)
	}

	return probes, nil
}

func (m *Metric) probes(name string, services []service.Service) []artifact.Query {
	series := name
	if m.Type == Histogram {
		series += "_count"
	}

	probe := func(selector, origin string) artifact.Query {
		return artifact.Query{
			Language: artifact.PromQL,
			Expr:     fmt.Sprintf("present_over_time(%s%s[%s])", series, selector, liveWindow),
			Origin:   origin,
		}
	}

	if m.EmittedBy == nil {
		return []artifact.Query{probe("", name+" (stack)")}
	}

	scope, perProcess := processSelectors[m.Source]

	var probes []artifact.Query

	for i := range services {
		svc := &services[i]

		for _, process := range svc.Processes {
			if !m.EmittedBy(svc, process) {
				continue
			}

			if !perProcess {
				return []artifact.Query{probe("", name+" (emitted by "+process.Name()+")")}
			}

			probes = append(probes, probe(scope(process.Name()), name+" of "+process.Name()))
		}
	}

	return probes
}
