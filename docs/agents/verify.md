# Verify

## Static

- `make -C frontend format` (commit what it changes)
- `make -C backend lint`
- `make -C backend o11y.lint`
- `make -C backend proto.lint`
- `make -C frontend lint`
- `make -C frontend typecheck`
- `docker run --rm -v "$PWD:/work:ro" -w /work dshanley/vacuum:v0.33.0 lint --details --no-banner --no-style --no-update-check openapi/*.yaml`

## Tests

- `docker compose -f backend/docker-compose.yml up -d --wait postgres`
- `make -C backend test`
- `make -C frontend test`

## Runtime

- `tilt up`, then the health check in `docs/agents/environment.md`
- `make -C frontend test.e2e`
- `make -C backend o11y.check-live`

Run when the change touches a service, the web app, the Tiltfiles or the observability config.

## Exclusions

- `make init` and `make clean-example`: CI only, they rewrite the template in place.
