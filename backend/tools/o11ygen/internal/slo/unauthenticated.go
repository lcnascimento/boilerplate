package slo

import (
	"strconv"
	"strings"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

type RatioQuery struct {
	Series   string
	Selector string
	Matcher  string
}

func Unauthenticated(runtime service.Runtime, processes []service.Process) RatioQuery {
	names := make([]string, 0, len(processes))
	for _, p := range processes {
		names = append(names, p.Name())
	}

	if runtime == service.Node {
		return RatioQuery{
			Series:   catalog.TracesSpanMetricsCalls,
			Selector: oneOf("service", names) + `, span_kind="SPAN_KIND_SERVER"`,
			Matcher:  `http_status_code="401"`,
		}
	}

	jobs := make([]string, 0, len(names))
	for _, name := range names {
		jobs = append(jobs, config.Namespace+"/"+name)
	}

	return RatioQuery{
		Series:   catalog.HTTPServerRequestTotal,
		Selector: oneOf("job", jobs) + `, http_route!~` + strconv.Quote(ProbeRoutes),
		Matcher:  `http_response_status_code="401"`,
	}
}

func oneOf(label string, values []string) string {
	if len(values) == 1 {
		return label + "=" + strconv.Quote(values[0])
	}

	return label + "=~" + strconv.Quote(strings.Join(values, "|"))
}
