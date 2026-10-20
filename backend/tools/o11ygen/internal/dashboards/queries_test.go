package dashboards_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/dashboards"
)

func TestQueries_ExtractsEveryPrometheusAndLokiQuery(t *testing.T) {
	board := `{
		"templating": {"list": [
			{"type": "datasource", "name": "prometheus_datasource", "query": "prometheus"},
			{"type": "query", "name": "service", "datasource": {"type": "prometheus", "uid": "${prometheus_datasource}"},
			 "query": "label_values(http_server_request_total, job)"},
			{"type": "query", "name": "topic", "datasource": {"type": "prometheus", "uid": "${prometheus_datasource}"},
			 "query": {"query": "label_values(kafka_topic_partition_current_offset, topic)", "refId": "topic-var"}},
			{"type": "query", "name": "container", "datasource": {"type": "loki", "uid": "${loki_datasource}"},
			 "query": "label_values(container)"}
		]},
		"annotations": {"list": [
			{"datasource": {"type": "grafana", "uid": "-- Grafana --"}, "name": "Annotations & Alerts"}
		]},
		"panels": [
			{"id": 1, "title": "Rate", "datasource": {"type": "prometheus", "uid": "${prometheus_datasource}"},
			 "targets": [{"refId": "A", "expr": "sum(rate(http_server_request_total[$__rate_interval]))"}]},
			{"id": 2, "title": "Row", "type": "row", "panels": [
				{"id": 3, "title": "Errors", "datasource": {"type": "loki", "uid": "${loki_datasource}"},
				 "targets": [{"refId": "A", "expr": "{service_name=~\".+\"}"}]}
			]},
			{"id": 4, "title": "Mixed", "datasource": {"type": "prometheus", "uid": "${prometheus_datasource}"},
			 "targets": [{"refId": "A", "datasource": {"type": "loki", "uid": "${loki_datasource}"}, "expr": "{container=\"kafka\"}"}]},
			{"id": 5, "title": "Traces", "datasource": {"type": "tempo", "uid": "${tempo_datasource}"},
			 "targets": [{"refId": "A", "query": "{}"}]}
		]
	}`

	queries, err := dashboards.Kind().Queries([]byte(board))

	require.NoError(t, err)
	assert.Equal(t, []artifact.Query{
		{Language: artifact.PromQL, Expr: "group by (job) (http_server_request_total)", Origin: "variable service"},
		{Language: artifact.PromQL, Expr: "group by (topic) (kafka_topic_partition_current_offset)", Origin: "variable topic"},
		{Language: artifact.LogQL, Expr: `sum by (container) (count_over_time({container=~".+"}[1m]))`, Origin: "variable container"},
		{Language: artifact.PromQL, Expr: "sum(rate(http_server_request_total[$__rate_interval]))", Origin: `panel 1 "Rate" target A`},
		{Language: artifact.LogQL, Expr: `{service_name=~".+"}`, Origin: `panel 3 "Errors" target A`},
		{Language: artifact.LogQL, Expr: `{container="kafka"}`, Origin: `panel 4 "Mixed" target A`},
	}, queries)
}

func TestQueries_RejectsADatasourceWithoutAQueryLanguage(t *testing.T) {
	board := `{"panels": [{"id": 1, "title": "SQL", "datasource": {"type": "postgres", "uid": "pg"},
		"targets": [{"refId": "A", "expr": "select 1"}]}]}`

	_, err := dashboards.Kind().Queries([]byte(board))

	require.ErrorContains(t, err, `datasource type "postgres" has no query language`)
}
