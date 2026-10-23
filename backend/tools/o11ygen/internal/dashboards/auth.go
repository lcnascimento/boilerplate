package dashboards

import (
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

func unauthenticatedRatio(runtime service.Runtime, processes []service.Process) string {
	q := slo.Unauthenticated(runtime, processes)
	sum := func(matchers string) string {
		return "sum (rate(" + q.Series + "{" + matchers + "}" + rate + "))"
	}

	return "(" + sum(q.Selector+", "+q.Matcher) + " or vector(0)) / " + sum(q.Selector)
}

func countedBy(series string, processes []service.Process, labels ...string) query {
	by := groupBy(labels...)

	return query{
		expr:   "sum by (" + by.by + ") (rate(" + series + "{" + job(processes) + "}" + rate + "))",
		legend: by.legend,
	}
}

func goAuthSection(processes []service.Process) section {
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
				expr: unauthenticatedRatio(service.Go, processes),
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
				expr: unauthenticatedRatio(service.Node, processes),
			}},
		},
	}}
}
