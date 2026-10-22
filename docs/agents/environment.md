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
