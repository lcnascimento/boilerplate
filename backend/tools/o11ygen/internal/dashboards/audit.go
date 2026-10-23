package dashboards

import (
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/rules"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

func identityChangesSection(svc *service.Service) (section, bool) {
	processes := declaring(svc, service.Process.IdentityChanges)
	if len(processes) == 0 {
		return section{}, false
	}

	age := catalog.IAMIdentityChangeOldestPendingAge + "{" + job(processes) + "}"
	threshold := rules.IdentityChangeBacklogThreshold.Seconds()

	return section{title: "Mudanças de identidade", panels: []panel{
		&single{
			title:       "Mudança pendente mais antiga",
			description: "Idade da mudança de identidade mais antiga ainda não traduzida em Integration Event; 0 sem pendentes.",
			unit:        "s",
			width:       thirdWidth,
			queries:     []query{{expr: "max(" + age + ")"}},
			steps:       []step{{color: "green"}, at(threshold, "red")},
		},
		&graph{
			title: "Idade da mudança pendente mais antiga",
			description: "Idade da mudança de identidade pendente mais antiga, por processo; a linha é o limite do alerta " +
				"IdentityChangeBacklogHigh.",
			unit:    "s",
			width:   thirdWidth,
			queries: []query{{expr: "max by (job) (" + age + ")", legend: "{{job}}"}},
			sloLine: &threshold,
		},
		&graph{
			title:       "Traduzidas por tipo de evento",
			description: "Mudanças de identidade traduzidas em Integration Events por segundo, pelo tipo do evento.",
			unit:        "ops",
			width:       thirdWidth,
			queries:     []query{countedBy(catalog.IAMIdentityChangeTranslated, processes, "cloudevents_event_type")},
		},
	}}, true
}

func auditLogSection(svc *service.Service) (section, bool) {
	processes := declaring(svc, service.Process.AuditLog)
	if len(processes) == 0 {
		return section{}, false
	}

	return section{title: "Audit Log", panels: []panel{
		&graph{
			title:       "Audit Entries por tipo de evento",
			description: "Audit Entries gravadas no intervalo, pelo tipo do evento; uma reentrega conta de novo.",
			unit:        "short",
			width:       fullWidth,
			queries: []query{{
				expr: "sum by (cloudevents_event_type) (increase(" + catalog.AuditEntriesRecorded + "{" + job(processes) + "}" +
					rate + "))",
				legend: "{{cloudevents_event_type}}",
			}},
		},
	}}, true
}

func declaring(svc *service.Service, option func(service.Process) bool) []service.Process {
	var out []service.Process

	for _, p := range svc.Processes {
		if option(p) {
			out = append(out, p)
		}
	}

	return out
}
