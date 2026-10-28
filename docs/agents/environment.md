# Environment

## Setup

- `make setup`

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

## Ephemeral environment

A worktree runs the application isolated from `tilt up` and from other worktrees, reporting to the shared dev o11y stack. Run every target from the worktree; each fails in the main checkout, whose environment is `local`.

- `make env.up`: starts the compose project `<env>` (Postgres, Kafka, Schema Registry, with host ports from Docker and no named volume), bootstraps it with the targets Tilt runs, then builds and starts every service and the web app in the background, on free ports. Returns once every `/health` answers, and prints `.env.ephemeral`. Running it again restarts the processes. Fails when the o11y stack is down: start it with `tilt up` in the main checkout.
- `make env.down`: stops the processes and runs `docker compose -p <env> down -v`; idempotent.
- `make env.status`: prints `.env.ephemeral` and the state of the processes and containers.
- `make env.prune`: removes the ephemeral compose projects whose worktree is gone; runs from any checkout.
- `<env>` is the worktree directory's basename, lowercased, with any character outside `[a-z0-9-]` replaced by `-`. It names the compose project and `deployment.environment`.
- `.env.ephemeral`, at the worktree root, holds the ports and URLs (`WEB_URL`, `IAM_DATABASE_URL`, `KAFKA_BROKERS`, ...); `make test.frontend.e2e` loads it, so it runs against the ephemeral environment.
- Process logs and PIDs: `.ephemeral/<process>.log` and `.ephemeral/<process>.pid`.
- Signals: filter by `deployment_environment="<env>"` (services) or `compose_project="<env>"` (container logs in Loki). The Alloy exporters still scrape only the dev Postgres and Kafka, so `pg_*` and `kafka_*` have no ephemeral series.

## Teardown

- `make env.down`

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

Every service metric carries `deployment_environment` (`local` under Tilt) next to `job` (`boilerplate/<service>`); filter by it once several environments share the stack: `{job="boilerplate/<service>", deployment_environment="local"}`.

Loki stream labels are `service_namespace` (`boilerplate`), `service_name` and `deployment_environment` for the services and the web app, and `service_name`, `compose_project` (`backend`) and `compose_service` for the infrastructure containers; `trace_id`, `scope_name`, `severity_number` and log attributes are structured metadata, filtered after the selector: `{service_namespace="boilerplate", service_name="<service>"} | trace_id="<trace_id>"`.

Without the MCP server, query the backends over HTTP:

- Logs: `curl -sG http://localhost:3100/loki/api/v1/query_range -H 'X-Loki-Response-Encoding-Flags: categorize-labels' --data-urlencode 'query={service_namespace="boilerplate", service_name="<service>"} |= "<text>"' --data-urlencode 'since=1h' --data-urlencode 'limit=50'`; each value is `[ts, line, {"structuredMetadata": {"trace_id": ...}}]`.
- Traces: `curl -s http://localhost:3200/api/traces/<trace_id>`
- Metrics: `curl -sG http://localhost:9090/api/v1/query --data-urlencode 'query=<promql>'`
