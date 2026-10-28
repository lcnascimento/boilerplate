package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const (
	identityChangesSection = "Mudanças de identidade"
	auditLogSection        = "Audit Log"
)

var accountsService = service.Service{
	Name: "accounts",
	Processes: []service.Process{
		service.Daemon("accounts-worker", service.Worker).WithDBPool().WithIdentityChanges(),
		service.Daemon("accounts-audit", service.Consumer).WithConsumerGroup("audit").WithAuditLog(),
	},
}

func generateAccounts(t *testing.T) (string, board) {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{accountsService}))

	return root, readBoard(t, root, "accounts.json")
}

func TestGenerate_IdentityChangesAndAuditLogAddTheirRowsAfterTheComponents(t *testing.T) {
	_, accounts := generateAccounts(t)

	assert.Equal(t, []string{
		"SLOs e alertas", "consumer", "worker", identityChangesSection, auditLogSection, "Runtime", "Banco", "Logs",
	}, rowTitles(&accounts))
}

func TestGenerate_IdentityChangesShowTheBacklogAgeAndTheTranslationsByEventType(t *testing.T) {
	_, accounts := generateAccounts(t)

	age := `iam_identity_change_oldest_pending_age_seconds{job=~"boilerplate/(accounts-worker)", deployment_environment="$environment"}`
	assert.Equal(t, map[string][]string{
		"Mudança pendente mais antiga":          {"max(" + age + ") => "},
		"Idade da mudança pendente mais antiga": {"max by (job) (" + age + ") => {{job}}"},
		"Traduzidas por tipo de evento": {
			`sum by (cloudevents_event_type) (rate(iam_identity_change_translated_total{job=~"boilerplate/(accounts-worker)", deployment_environment="$environment"}` +
				`[$__rate_interval])) => {{cloudevents_event_type}}`,
		},
	}, legendsByTitle(accounts.section(t, identityChangesSection).panels,
		"Mudança pendente mais antiga", "Idade da mudança pendente mais antiga", "Traduzidas por tipo de evento"))
}

func TestGenerate_AuditLogCountsTheEntriesByEventType(t *testing.T) {
	_, accounts := generateAccounts(t)

	assert.Equal(t, map[string][]string{
		"Audit Entries por tipo de evento": {
			`sum by (cloudevents_event_type) (increase(audit_entries_recorded_total{job=~"boilerplate/(accounts-audit)", deployment_environment="$environment"}` +
				`[$__rate_interval])) => {{cloudevents_event_type}}`,
		},
	}, legendsByTitle(accounts.section(t, auditLogSection).panels, "Audit Entries por tipo de evento"))
}

func TestGenerate_LeavesOutTheIdentityChangesAndAuditLogRowsWithoutTheirOptions(t *testing.T) {
	_, orders := generateOrders(t)

	assert.NotContains(t, rowTitles(&orders), identityChangesSection)
	assert.NotContains(t, rowTitles(&orders), auditLogSection)
}

func TestGenerate_AlertsWhenTheOldestIdentityChangeWaitsMoreThanFiveMinutes(t *testing.T) {
	root, _ := generateAccounts(t)

	exprs := map[string]string{}

	for _, group := range readRules(t, filepath.Join(root, rulesDir, "accounts.yaml")).Groups {
		for _, rule := range group.Rules {
			if rule.Alert == "IdentityChangeBacklogHigh" {
				exprs[rule.Labels["service"]] = rule.Expr
			}
		}
	}

	assert.Equal(t, map[string]string{
		"accounts-worker": `max by (deployment_environment) (iam_identity_change_oldest_pending_age_seconds{job="boilerplate/accounts-worker"}) > 300`,
	}, exprs)
}

func TestGenerate_RejectsIdentityChangesAndAuditLogOnAProcessWithoutTheirComponent(t *testing.T) {
	for _, tc := range []struct {
		process service.Process
		want    error
	}{
		{service.Daemon("accounts-api", service.API).WithIdentityChanges(), service.ErrIdentityChangesWithoutWorker},
		{service.OneShot("accounts-migrate").WithIdentityChanges(), service.ErrIdentityChangesWithoutWorker},
		{service.Daemon("accounts-worker", service.Worker).WithAuditLog(), service.ErrAuditLogWithoutConsumer},
		{service.OneShot("accounts-migrate").WithAuditLog(), service.ErrAuditLogWithoutConsumer},
	} {
		err := generate(t.TempDir(), []service.Service{{Name: "accounts", Processes: []service.Process{tc.process}}})

		require.ErrorIs(t, err, tc.want)
		require.Equal(t, tc.process.Name(), errors.Attributes(err)["process"])
	}
}
