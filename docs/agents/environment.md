# Environment

## Start

- `tilt up`, or `TILT_STACK=minimal tilt up` without the observability stack and the Kafka console
- Services:
  - `postgres`
  - `postgres-extensions`
  - `kafka`
  - `schema-registry`
  - `kafka-console`
  - `alloy`
  - `prometheus`
  - `loki`
  - `tempo`
  - `pyroscope`
  - `grafana`

## Stop

- `tilt down`

## Variables

- Copy `.env.example` to `.env`; `direnv` loads it.
- `LINEAR_API_KEY`
- `GRAFANA_SERVICE_ACCOUNT_TOKEN`: `make -C backend grafana.token` writes it (Viewer service account `MCP`), and `tilt up` runs it once Grafana is up; restart Claude Code after a new one.
- `make -C backend env` creates `backend/.env` and each `backend/services/*/.env` from their examples; `tilt up` runs it.

## Health check

- `tilt wait --for=condition=Ready 'uiresource/(Tiltfile)' --timeout=2m`, then every resource Ready in `tilt get uiresource` (UI at http://localhost:10350).

## Local signals

Query them through the `grafana` MCP server (`mcp__grafana__*`, needs `GRAFANA_SERVICE_ACCOUNT_TOKEN`);

- Logs: Loki, `query_loki_logs`. `tilt logs -f <resource>` for a process that fails before it exports logs.
- Metrics: Prometheus, `query_prometheus`
- Traces: Tempo, `search_tempo_traces`
- Profiles: Pyroscope, `query_pyroscope`
- Dashboards: Grafana, `search_dashboards`

401 from `mcp__grafana__*` means `GRAFANA_SERVICE_ACCOUNT_TOKEN` is invalid: run `make -C backend grafana.token`, then restart Claude Code.

Loki stream selectors take `service_namespace` (`boilerplate`) and `service_name`; `trace_id`, `scope_name` and log attributes are structured metadata, filtered after the selector: `{service_namespace="boilerplate", service_name="<service>"} | trace_id="<trace_id>"`.

Without the MCP server, query the backends over HTTP:

- Logs: `curl -sG http://localhost:3100/loki/api/v1/query_range --data-urlencode 'query={service_namespace="boilerplate", service_name="<service>"} |= "<text>"' --data-urlencode 'since=1h' --data-urlencode 'limit=50'`; each entry's structured metadata carries its `trace_id`.
- Traces: `curl -s http://localhost:3200/api/traces/<trace_id>`
- Metrics: `curl -sG http://localhost:9090/api/v1/query --data-urlencode 'query=<promql>'`
