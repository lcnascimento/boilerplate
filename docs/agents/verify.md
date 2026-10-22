# Verify

## Static

- `make format` (commit what it changes)
- `make lint`

## Tests

- `make test`

## Runtime

- `tilt up`, then the health check in `docs/agents/environment.md`
- `make test.frontend.e2e`
- `make -C backend o11y.check-live`

Run when the change touches a service, the web app, the Tiltfiles or the observability config.

## Exclusions

- `make init` and `make clean-example`: CI only, they rewrite the template in place.
