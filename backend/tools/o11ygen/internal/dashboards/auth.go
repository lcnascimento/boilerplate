package dashboards

import (
	"strconv"
	"strings"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/slo"
)

const authSectionTitle = "Autenticação e autorização"

func authSection(svc *service.Service) (section, bool) {
	var authenticated []service.Process

	for _, p := range svc.Processes {
		if p.ActorAuth() {
			authenticated = append(authenticated, p)
		}
	}

	if len(authenticated) == 0 {
		return section{}, false
	}

	return runtimeDashboards[svc.Runtime].auth(authenticated), true
}

func ratio(series, selector, matcher string) string {
	sum := func(matchers string) string {
		return "sum (rate(" + series + "{" + matchers + "}" + rate + "))"
	}

	return "(" + sum(selector+", "+matcher) + " or vector(0)) / " + sum(selector)
}

func countedBy(series string, processes []service.Process, labels ...string) query {
	by := groupBy(labels...)

	return query{
		expr:   "sum by (" + by.by + ") (rate(" + series + "{" + job(processes) + "}" + rate + "))",
		legend: by.legend,
	}
}

func goAuthSection(processes []service.Process) section {
	requests := job(processes) + `, http_route!~` + strconv.Quote(slo.ProbeRoutes)

	return section{title: authSectionTitle, panels: []panel{
		&graph{
			title:       "401 por motivo",
			description: "Requisições que o middleware de Actor recusou por segundo, pelo motivo da recusa.",
			unit:        "reqps",
			width:       thirdWidth,
			queries:     []query{countedBy(catalog.IAMActorAuthenticationFailures, processes, "reason")},
		},
		&graph{
			title:       "Taxa de 401",
			description: "Fração das requisições respondidas com 401, fora as de health e readiness; é o que o alerta AuthenticationFailuresHigh observa.",
			unit:        "percentunit",
			width:       thirdWidth,
			queries: []query{{
				expr: ratio(catalog.HTTPServerRequestTotal, requests, `http_response_status_code="401"`),
			}},
		},
		&graph{
			title:       "403 por Permission e Role",
			description: "Ações negadas por segundo porque a Role do Actor não tem a Permission.",
			unit:        "ops",
			width:       thirdWidth,
			queries:     []query{countedBy(catalog.IAMActorAuthorizationDenials, processes, "permission", "role")},
		},
	}}
}

func nodeAuthSection(processes []service.Process) section {
	server := `service=~` + strconv.Quote(strings.Join(processNames(processes), "|")) + `, span_kind="SPAN_KIND_SERVER"`

	return section{title: authSectionTitle, panels: []panel{
		&graph{
			title:       "Falhas de login por motivo",
			description: "Sign-ins e sign-ups recusados por segundo, pelo motivo da recusa.",
			unit:        "ops",
			width:       halfWidth,
			queries:     []query{countedBy(catalog.IAMLoginFailures, processes, "reason")},
		},
		&graph{
			title:       "Taxa de 401",
			description: "Fração das requisições respondidas com 401; é o que o alerta AuthenticationFailuresHigh observa.",
			unit:        "percentunit",
			width:       halfWidth,
			queries: []query{{
				expr: ratio(catalog.TracesSpanMetricsCalls, server, `http_status_code="401"`),
			}},
		},
	}}
}
