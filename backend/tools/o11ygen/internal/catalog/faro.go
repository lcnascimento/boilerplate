package catalog

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

type FaroKind string

const (
	FaroMeasurement FaroKind = "measurement"
	FaroEvent       FaroKind = "event"
	FaroException   FaroKind = "exception"
)

type WebVital string

const (
	LCP  WebVital = "lcp"
	INP  WebVital = "inp"
	CLS  WebVital = "cls"
	FCP  WebVital = "fcp"
	TTFB WebVital = "ttfb"
)

func (v WebVital) ValueLabel() string { return "value_" + string(v) }

const (
	FaroTypeLabel      = "type"
	FaroEventNameLabel = "event_name"
	FaroWebVitalsType  = "web-vitals"
	FaroPageLoadEvent  = "faro.performance.navigation"

	faroKindLabel = "kind"
)

type faroStream struct {
	labels   []string
	presence Presence
}

var faroStreams = map[FaroKind]faroStream{
	FaroMeasurement: {labels: []string{
		FaroTypeLabel, LCP.ValueLabel(), INP.ValueLabel(), CLS.ValueLabel(), FCP.ValueLabel(), TTFB.ValueLabel(),
	}},
	FaroEvent:     {labels: []string{FaroEventNameLabel}},
	FaroException: {labels: []string{FaroTypeLabel}, presence: PresenceOnEvent},
}

func faroLabels() []string {
	out := []string{faroKindLabel}
	for _, stream := range faroStreams {
		out = append(out, stream.labels...)
	}

	return out
}

func FaroStream(kind FaroKind, processes []string) string {
	return `{service_namespace=` + strconv.Quote(config.Namespace) +
		`, service_name=~` + strconv.Quote(strings.Join(processes, "|")) + `} | ` +
		faroKindLabel + `=` + strconv.Quote(string(kind))
}

func faroProbes(services []service.Service, window string) []artifact.Query {
	kinds := make([]FaroKind, 0, len(faroStreams))
	for kind, stream := range faroStreams {
		if stream.presence == PresenceAlways {
			kinds = append(kinds, kind)
		}
	}

	slices.Sort(kinds)

	var probes []artifact.Query

	for i := range services {
		for _, process := range services[i].Processes {
			if !process.Runs(service.Browser) {
				continue
			}

			for _, kind := range kinds {
				probes = append(probes, artifact.Query{
					Language: artifact.LogQL,
					Expr: fmt.Sprintf("sum(count_over_time(%s [%s]))",
						FaroStream(kind, []string{process.Name()}), window),
					Origin: "faro " + string(kind) + " of " + process.Name(),
				})
			}
		}
	}

	return probes
}
