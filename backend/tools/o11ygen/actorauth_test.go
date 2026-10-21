package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const authSection = "Autenticação e autorização"

var paymentsService = service.Service{
	Name: "payments",
	Processes: []service.Process{
		service.Daemon("payments-api", service.API).WithActorAuth().WithDBPool(),
		service.Daemon("payments-worker", service.Worker),
	},
}

var shopService = service.Service{
	Name:      "shop",
	Runtime:   service.Node,
	Processes: []service.Process{service.Daemon("shop", service.API, service.Browser).WithActorAuth()},
}

func rowTitles(b *board) []string {
	sections := b.sections()
	rows := make([]string, 0, len(sections))

	for i := range sections {
		rows = append(rows, sections[i].row.Title)
	}

	return rows
}

func TestGenerate_ActorAuthAddsTheAuthenticationRowAfterTheComponentsOfAGoService(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{paymentsService}))

	payments := readBoard(t, root, "payments.json")

	assert.Equal(t, []string{"SLOs e alertas", "api", "worker", authSection, "Runtime", "Banco", "Logs"}, rowTitles(&payments))

	requests := `http_server_request_total{job=~"boilerplate/(payments-api)", http_route!~"/ping|/health"`
	assert.Equal(t, map[string][]string{
		"401 por motivo": {
			`sum by (reason) (rate(iam_actor_authentication_failures_total{job=~"boilerplate/(payments-api)"}[$__rate_interval])) => {{reason}}`,
		},
		"Taxa de 401": {
			`(sum (rate(` + requests + `, http_response_status_code="401"}[$__rate_interval])) or vector(0)) / sum (rate(` +
				requests + `}[$__rate_interval])) => `,
		},
		"403 por Permission e Role": {
			`sum by (permission, role) (rate(iam_actor_authorization_denials_total{job=~"boilerplate/(payments-api)"}[$__rate_interval])) => {{permission}} {{role}}`,
		},
	}, legendsByTitle(payments.section(t, authSection).panels, "401 por motivo", "Taxa de 401", "403 por Permission e Role"))
}

func TestGenerate_ActorAuthShowsTheLoginFailuresAndThe401RateOfANodeService(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{shopService}))

	shop := readBoard(t, root, "shop.json")

	assert.Equal(t, []string{"Alertas", "api", "browser", authSection, "Runtime", "Logs"}, rowTitles(&shop))

	calls := `traces_spanmetrics_calls_total{service=~"shop", span_kind="SPAN_KIND_SERVER"`
	assert.Equal(t, map[string][]string{
		"Falhas de login por motivo": {
			`sum by (reason) (rate(iam_login_failures_total{job=~"boilerplate/(shop)"}[$__rate_interval])) => {{reason}}`,
		},
		"Taxa de 401": {
			`(sum (rate(` + calls + `, http_status_code="401"}[$__rate_interval])) or vector(0)) / sum (rate(` +
				calls + `}[$__rate_interval])) => `,
		},
	}, legendsByTitle(shop.section(t, authSection).panels, "Falhas de login por motivo", "Taxa de 401", "403 por Permission e Role"))
}

func TestGenerate_LeavesOutTheAuthenticationRowWithoutActorAuth(t *testing.T) {
	_, orders := generateOrders(t)
	_, storefront := generateStorefront(t)

	assert.NotContains(t, rowTitles(&orders), authSection)
	assert.NotContains(t, rowTitles(&storefront), authSection)
}

func TestGenerate_AlertsWhenThe401RateOfAnActorAuthProcessStaysHigh(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{paymentsService, shopService}))

	rules := generatedRules(t, root)
	alerts := map[string]string{}

	for _, file := range []string{"payments.yaml", "shop.yaml"} {
		for _, group := range rules[file].Groups {
			for _, rule := range group.Rules {
				if rule.Alert == "AuthenticationFailuresHigh" {
					alerts[rule.Labels["service"]] = rule.Expr
				}
			}
		}
	}

	requests := `http_server_request_total{job="boilerplate/payments-api", http_route!~"/ping|/health"`
	calls := `traces_spanmetrics_calls_total{service="shop", span_kind="SPAN_KIND_SERVER"`
	assert.Equal(t, map[string]string{
		"payments-api": `sum(rate(` + requests + `, http_response_status_code="401"}[5m])) / sum(rate(` + requests +
			`}[5m])) > 0.1`,
		"shop": `sum(rate(` + calls + `, http_status_code="401"}[5m])) / sum(rate(` + calls + `}[5m])) > 0.1`,
	}, alerts)
}

func TestGenerate_RejectsActorAuthOnAProcessWithoutAnAPI(t *testing.T) {
	for _, process := range []service.Process{
		service.Daemon("payments-worker", service.Worker).WithActorAuth(),
		service.OneShot("payments-migrate").WithActorAuth(),
	} {
		err := generate(t.TempDir(), []service.Service{{Name: "payments", Processes: []service.Process{process}}})

		require.ErrorContains(t, err, `process "`+process.Name()+`" declares actor auth but serves no api`)
	}
}
