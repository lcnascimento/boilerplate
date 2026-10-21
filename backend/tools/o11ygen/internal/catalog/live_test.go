package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

func exprs(t *testing.T, declared []service.Service) []string {
	t.Helper()

	probes, err := catalog.LiveProbes(declared)
	require.NoError(t, err)

	out := make([]string, 0, len(probes))
	for _, probe := range probes {
		out = append(out, probe.Expr)
	}

	return out
}

func TestLiveProbes_WithoutServicesExpectOnlyTheStack(t *testing.T) {
	assert.Equal(t, []string{
		`present_over_time(job:series:count[15m])`,
		`present_over_time(kafka_brokers[15m])`,
		`present_over_time(pg_client_connections_count[15m])`,
		`present_over_time(pg_database_size_bytes[15m])`,
		`present_over_time(pg_locks_count[15m])`,
		`present_over_time(pg_settings_max_connections[15m])`,
		`present_over_time(pg_stat_activity_count[15m])`,
		`present_over_time(pg_stat_activity_max_tx_duration[15m])`,
		`present_over_time(pg_stat_database_blks_hit[15m])`,
		`present_over_time(pg_stat_database_blks_read[15m])`,
		`present_over_time(pg_stat_database_deadlocks[15m])`,
		`present_over_time(pg_stat_database_xact_commit[15m])`,
		`present_over_time(pg_stat_database_xact_rollback[15m])`,
		`present_over_time(pg_stat_statements_calls_total[15m])`,
		`present_over_time(pg_stat_statements_rows_total[15m])`,
		`present_over_time(pg_stat_statements_seconds_total[15m])`,
		`present_over_time(pg_up[15m])`,
	}, exprs(t, nil))
}

func TestLiveProbes_ExpectWhatEachDeclaredProcessEmits(t *testing.T) {
	declared := []service.Service{{
		Name: "orders",
		Processes: []service.Process{
			service.Daemon("orders-api", service.API).WithDBPool(),
			service.OneShot("orders-migrate"),
		},
	}}

	got := exprs(t, declared)

	for _, want := range []string{
		`present_over_time(target_info{job="boilerplate/orders-api"}[15m])`,
		`present_over_time(http_server_request_total{job="boilerplate/orders-api"}[15m])`,
		`present_over_time(http_server_request_duration_seconds_count{job="boilerplate/orders-api"}[15m])`,
		`present_over_time(pgxpool_acquired_connections{job="boilerplate/orders-api"}[15m])`,
		`present_over_time(service:log_records:rate1m[15m])`,
		`present_over_time(pg_up[15m])`,
	} {
		assert.Contains(t, got, want)
	}

	for _, unwanted := range []string{
		`present_over_time(target_info{job="boilerplate/orders-migrate"}[15m])`,
		`present_over_time(messaging_client_sent_messages_total{job="boilerplate/orders-api"}[15m])`,
		`present_over_time(kafka_consumergroup_current_offset[15m])`,
		`present_over_time(service:log_records:rate1m{service="orders-api"}[15m])`,
		`present_over_time(service:log_records_critical:rate1m[15m])`,
	} {
		assert.NotContains(t, got, unwanted)
	}
}

func TestLiveProbes_ExpectTheConsumerGroupSeriesOnceAConsumerIsDeclared(t *testing.T) {
	declared := []service.Service{{
		Name: "billing",
		Processes: []service.Process{
			service.Daemon("billing-consumer", service.Consumer).WithConsumerGroup("billing"),
		},
	}}

	got := exprs(t, declared)

	assert.Contains(t, got, `present_over_time(kafka_consumergroup_uncommitted_offsets[15m])`)
	assert.Contains(t, got, `present_over_time(messaging_process_duration_seconds_count{job="boilerplate/billing-consumer"}[15m])`)
	assert.NotContains(t, got, `present_over_time(http_server_request_total{job="boilerplate/billing-consumer"}[15m])`)
}

func TestLiveProbes_ExpectTheOutboxOfTheServiceAWorkerBelongsTo(t *testing.T) {
	declared := []service.Service{
		{Name: "orders", Processes: []service.Process{service.Daemon("orders-worker", service.Worker)}},
		{Name: "billing", Processes: []service.Process{service.Daemon("billing-api", service.API)}},
	}

	got := exprs(t, declared)

	for _, want := range []string{
		`present_over_time(orders_outbox_oldest_pending_age_seconds{job="boilerplate/orders-worker"}[15m])`,
		`present_over_time(orders_outbox_purged_messages_total{job="boilerplate/orders-worker"}[15m])`,
		`present_over_time(orders_outbox_purge_runs_total{job="boilerplate/orders-worker"}[15m])`,
	} {
		assert.Contains(t, got, want)
	}

	for _, probe := range got {
		assert.NotContains(t, probe, "billing_outbox_")
	}
}

func TestLiveProbes_ExpectTheNodeRuntimeAndSpanMetricsOfANodeService(t *testing.T) {
	declared := []service.Service{{
		Name:      "web",
		Runtime:   service.Node,
		Processes: []service.Process{service.Daemon("web", service.API, service.Browser)},
	}}

	got := exprs(t, declared)

	for _, want := range []string{
		`present_over_time(target_info{job="boilerplate/web"}[15m])`,
		`present_over_time(nodejs_eventloop_utilization_ratio{job="boilerplate/web"}[15m])`,
		`present_over_time(v8js_gc_duration_seconds_count{job="boilerplate/web"}[15m])`,
		`present_over_time(traces_spanmetrics_calls_total{service="web"}[15m])`,
		`present_over_time(traces_spanmetrics_latency_count{service="web"}[15m])`,
		`sum(count_over_time({service_namespace="boilerplate", service_name=~"web"} | kind="event" [15m]))`,
		`sum(count_over_time({service_namespace="boilerplate", service_name=~"web"} | kind="measurement" [15m]))`,
	} {
		assert.Contains(t, got, want)
	}

	for _, unwanted := range []string{
		`present_over_time(http_server_request_total{job="boilerplate/web"}[15m])`,
		`present_over_time(go_goroutine_count{job="boilerplate/web"}[15m])`,
		`present_over_time(process_memory_usage_bytes{job="boilerplate/web"}[15m])`,
		`sum(count_over_time({service_namespace="boilerplate", service_name=~"web"} | kind="exception" [15m]))`,
	} {
		assert.NotContains(t, got, unwanted)
	}
}

func TestLiveProbes_StayInsideTheCatalog(t *testing.T) {
	declared := []service.Service{{
		Name: "every",
		Processes: []service.Process{
			service.Daemon("every-process", service.API, service.Consumer, service.Worker, service.Browser).
				WithConsumerGroup("every").WithDBPool(),
		},
	}}

	probes, err := catalog.LiveProbes(declared)
	require.NoError(t, err)

	for _, probe := range probes {
		assert.NoError(t, catalog.Check(probe), probe.Expr)
	}
}
