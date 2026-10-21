package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
)

func TestCheck_AcceptsQueriesInsideTheCatalog(t *testing.T) {
	queries := map[string]artifact.Query{
		"grafana variables and grouping by allowed labels": {
			Language: artifact.PromQL,
			Expr:     `sum by (http_route) (rate(http_server_request_total{job=~"$service", http_route!~"/health|/ping"}[$__rate_interval]))`,
		},
		"histogram buckets carry le": {
			Language: artifact.PromQL,
			Expr:     `histogram_quantile(0.99, sum by (le, http_route) (rate(http_server_request_duration_seconds_bucket[5m])))`,
		},
		"client metrics by peer, method and status": {
			Language: artifact.PromQL,
			Expr: `sum by (network_peer_address, http_request_method, http_response_status_code) ` +
				`(rate(http_client_request_total[5m])) / on (network_peer_address, http_request_method) group_left ` +
				`sum by (network_peer_address, http_request_method) (http_client_active_requests{server_address!="", server_port="80"})`,
		},
		"client latency buckets": {
			Language: artifact.PromQL,
			Expr:     `histogram_quantile(0.95, sum by (le, server_address) (rate(http_client_request_duration_seconds_bucket[5m])))`,
		},
		"labels of both sides of a binary operation": {
			Language: artifact.PromQL,
			Expr: `sum by (consumergroup, topic) (kafka_consumergroup_current_offset) ` +
				`/ on (topic) group_left sum by (topic) (kafka_topic_partition_current_offset)`,
		},
		"a selector without a metric name groups by target labels": {
			Language: artifact.PromQL,
			Expr:     `count by (job) ({__name__=~".+"})`,
		},
		"log labels and label filters": {
			Language: artifact.LogQL,
			Expr:     `sum by (service_name) (count_over_time({service_name=~"${service:regex}"} | detected_level=~"error|critical" [$__auto]))`,
		},
		"faro fields kept as structured metadata": {
			Language: artifact.LogQL,
			Expr: `quantile_over_time(0.75, {service_name=~"web"} | kind="measurement" | type="web-vitals" | ` +
				`value_lcp!="" | unwrap value_lcp [$__auto]) by ()`,
		},
		"the outbox of any service": {
			Language: artifact.PromQL,
			Expr: `sum by (outcome) (increase(orders_outbox_purge_runs_total{job="boilerplate/orders-worker"}[5m])) ` +
				`+ on () group_left max(billing_outbox_oldest_pending_age_seconds)`,
		},
		"container logs by the container name": {
			Language: artifact.LogQL,
			Expr:     `sum by (container_name) (count_over_time({service_name="kafka"} | detected_level="warn" [$__auto]))`,
		},
		"labels a label_format introduces": {
			Language: artifact.LogQL,
			Expr:     `sum by (service) (count_over_time({service_name=~".+"} | label_format service=service_name [1m]))`,
		},
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, catalog.Check(query))
		})
	}
}

func TestCheck_RejectsQueriesOutsideTheCatalog(t *testing.T) {
	queries := map[string]struct {
		query artifact.Query
		err   string
	}{
		"renamed metric": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `sum(rate(http_server_requests_total[5m]))`},
			err:   `metric "http_server_requests_total" is not in the catalog`,
		},
		"matcher on a label the metric does not allow": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `rate(http_server_request_total{url_path="/notes/1"}[5m])`},
			err:   `label "url_path" is not allowed on metric "http_server_request_total"`,
		},
		"route on a client metric": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `sum by (http_route) (rate(http_client_request_total[5m]))`},
			err:   `grouping by label "http_route", which the expression does not carry`,
		},
		"grouping by a label the metric does not allow": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `sum by (url_path) (rate(http_server_request_total[5m]))`},
			err:   `grouping by label "url_path", which the expression does not carry`,
		},
		"le outside a histogram bucket": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `sum by (le) (rate(http_server_request_total[5m]))`},
			err:   `grouping by label "le", which the expression does not carry`,
		},
		"histogram suffix on a counter": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `rate(http_server_request_total_bucket[5m])`},
			err:   `metric "http_server_request_total_bucket" is not in the catalog`,
		},
		"metric-specific label on a selector without a metric name": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `count by (http_route) ({__name__=~".+"})`},
			err:   `grouping by label "http_route", which the expression does not carry`,
		},
		"invalid PromQL": {
			query: artifact.Query{Language: artifact.PromQL, Expr: `sum(rate(http_server_request_total[5m])`},
			err:   `parsing PromQL`,
		},
		"log label outside the catalog": {
			query: artifact.Query{Language: artifact.LogQL, Expr: `{service_name="notes-api-frontend"} | user_id="42"`},
			err:   `label "user_id" is not a log label in the catalog`,
		},
		"grouping by a log label outside the catalog": {
			query: artifact.Query{Language: artifact.LogQL, Expr: `sum by (pod) (count_over_time({service_name=~".+"}[1m]))`},
			err:   `label "pod" is not a log label in the catalog`,
		},
		"the container label Alloy no longer writes": {
			query: artifact.Query{Language: artifact.LogQL, Expr: `{container="kafka"}`},
			err:   `label "container" is not a log label in the catalog`,
		},
		"parser that extracts unbounded labels": {
			query: artifact.Query{Language: artifact.LogQL, Expr: `{service_name="notes-api-frontend"} | json`},
			err:   `extracts labels the catalog cannot bound`,
		},
		"invalid LogQL": {
			query: artifact.Query{Language: artifact.LogQL, Expr: `{service_name="notes-api-frontend"`},
			err:   `parsing LogQL`,
		},
		"unknown language": {
			query: artifact.Query{Language: "TraceQL", Expr: `{}`},
			err:   `unknown query language "TraceQL"`,
		},
	}

	for name, tc := range queries {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, catalog.Check(tc.query), tc.err)
		})
	}
}
