package slo

import (
	"strconv"
	"strings"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	APILatencyThresholdSeconds      = 0.25
	ConsumerLatencyThresholdSeconds = 1
	OutboxFreshnessThresholdSeconds = 30
)

const (
	threeNines = 99.9
	twoNines   = 99
)

const ProbeRoutes = "/ping|/health"

type inheritedSLO struct {
	component service.Component
	slo       func(p service.Process, svc string) service.SLO
}

var goKitDefaults = []inheritedSLO{
	{component: service.API, slo: func(p service.Process, _ string) service.SLO {
		return service.SLO{
			Name:        "api-availability",
			Objective:   threeNines,
			Description: "Requisições HTTP de " + p.Name() + " respondidas sem 5xx, fora as de health e readiness.",
			SLI: service.EventsSLI{
				ErrorQuery: orZero(rate(catalog.HTTPServerRequestTotal, apiSelector(p), `http_response_status_code=~"5.."`),
					rate(catalog.HTTPServerRequestTotal, apiSelector(p))),
				TotalQuery: rate(catalog.HTTPServerRequestTotal, apiSelector(p)),
			},
		}
	}},
	{component: service.API, slo: func(p service.Process, _ string) service.SLO {
		return service.SLO{
			Name:        "api-latency",
			Objective:   twoNines,
			Description: "Requisições HTTP de " + p.Name() + " respondidas em até 250ms, fora as de health e readiness.",
			SLI:         slowerThan(catalog.HTTPServerRequestDuration, apiSelector(p), APILatencyThresholdSeconds),
		}
	}},
	{component: service.Consumer, slo: func(p service.Process, _ string) service.SLO {
		return service.SLO{
			Name:        "consumer-success",
			Objective:   threeNines,
			Description: "Tentativas de processamento de mensagens de " + p.Name() + " que terminam sem erro.",
			SLI: service.EventsSLI{
				ErrorQuery: orZero(rate(catalog.MessagingProcessDuration+"_count", job(p), `error_type!=""`),
					rate(catalog.MessagingProcessDuration+"_count", job(p))),
				TotalQuery: rate(catalog.MessagingProcessDuration+"_count", job(p)),
			},
		}
	}},
	{component: service.Consumer, slo: func(p service.Process, _ string) service.SLO {
		return service.SLO{
			Name:        "consumer-latency",
			Objective:   twoNines,
			Description: "Tentativas de processamento de mensagens de " + p.Name() + " que terminam em até 1s.",
			SLI:         slowerThan(catalog.MessagingProcessDuration, job(p), ConsumerLatencyThresholdSeconds),
		}
	}},
	{component: service.Worker, slo: func(p service.Process, svc string) service.SLO {
		return service.SLO{
			Name:      "outbox-freshness",
			Objective: twoNines,
			Description: "Minutos em que a mensagem pendente mais antiga do outbox de " + svc +
				" tem até 30s, segundo " + p.Name() + ".",
			SLI: service.RawSLI{ErrorRatioQuery: "avg_over_time((max " + byEnvironment + " (" + catalog.OutboxOldestPendingAge(svc) + "{" + job(p) +
				"}) > bool " + number(OutboxFreshnessThresholdSeconds) + ")[" + window + ":1m])"},
		}
	}},
}

const (
	window        = service.SLOWindow
	byEnvironment = "by (" + config.EnvironmentLabel + ")"
)

func job(p service.Process) string {
	return "job=" + strconv.Quote(config.Namespace+"/"+p.Name())
}

func apiSelector(p service.Process) string {
	return job(p) + ",http_route!~" + strconv.Quote(ProbeRoutes)
}

func rate(series string, matchers ...string) string {
	return "sum " + byEnvironment + " (rate(" + series + "{" + strings.Join(matchers, ",") + "}[" + window + "]))"
}

// orZero keeps the SLI of each environment at 0, instead of absent, while no bad event has happened in it yet: the
// series of a bad event only exists after the first one.
func orZero(query, total string) string {
	return query + " or 0 * " + total
}

func slowerThan(histogram, selector string, seconds float64) service.SLI {
	total := rate(histogram+"_count", selector)

	return service.EventsSLI{
		ErrorQuery: total + " - " + rate(histogram+"_bucket", selector, "le="+strconv.Quote(number(seconds))),
		TotalQuery: total,
	}
}

func number(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
