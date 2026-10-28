# Coding Standards

Judgement rules for review. A rule a linter could check belongs in the linter, not here.

## Hold every comment, config included, to `earn-every-comment`

Config files (Alloy, YAML, Makefile, Tiltfile, Dockerfile, CI workflows) carry the same comment bar as code, and are where narration and usage manuals pile up unnoticed.

Violation:

```make
# Run `make test` to run the tests. Requires postgres from docker compose.
# The target loops over every service and calls its own test target.
test:
```

Fix: delete both lines. Encode the postgres requirement as a target prerequisite; if humans need the usage instructions, write them as a document with `foundation:documentation`.

Source: `earn-every-comment`.

## Give every domain list one source

A closed set of domain values (components, roles, statuses, event names) is declared once at its authoritative source (a schema when one owns it, code otherwise); every other use derives from it, so adding a value cannot miss a copy.

Violation:

```go
for _, c := range []Component{API, Consumer, Worker, Browser} { ... }
// ...elsewhere in the package
names := []Component{API, Consumer, Worker, Browser}
```

Fix:

```go
var components = []Component{API, Consumer, Worker, Browser}
```

and range over `components` at each use. A test that enumerates the set on purpose, to pin it, is not a copy.

Source: `model-the-domain`.

## Log an error once, where it is handled

The code that handles an error logs it once: the application, or the port when the error originates there. Code that only passes an error on records it on the span and in the metrics (`error.type`) and does not log it, so each error reaches the logs, and the error tracking built on them, once.

Violation:

```go
func (a *Application) onError(ctx context.Context, err error) error {
	logger.ErrorBySeverity(ctx, err)

	return err
}

// ...and the event bus that called the application logs the same error again
end := func(err error) {
	span.SetAttributes(semconv.ErrorTypeKey.String(errorType(err)))
	logger.ErrorBySeverity(ctx, err)
}
```

Fix:

```go
end := func(err error) {
	span.SetAttributes(semconv.ErrorTypeKey.String(errorType(err)))
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
```

and log in the event bus only the errors it originates, such as a message that does not decode.

Source: `boundary-discipline`.

## Keep PII out of baggage

Baggage carries only pseudonymous and technical identifiers (`user.id`, `organization.id`, `note_id`, `cloudevents.*`), never an email, a name or user content: it is copied into every span, every log line and every tracked exception downstream.

Violation:

```go
ctx = baggage.ContextWithMembers(ctx,
	baggage.NewMember("user.id", user.ID.String()),
	baggage.NewMember("user.email", user.Email),
)
```

Fix:

```go
ctx = baggage.ContextWithMembers(ctx, baggage.NewMember("user.id", user.ID.String()))
```

## Hand a CI secret only to the step that uses it

- `${{ secrets.X }}` appears only in the `env:` of the step that needs it: never in a job or workflow `env:`, inside `run:` or as a command-line argument.
- The Doppler `ci` config holds only disposable credentials (test accounts, sandboxes, test-mode keys), never a `prd` value.
- A test that needs a secret skips when the variable is empty, so fork and Dependabot PRs, which get no repository secrets, stay green; under `CI_SECRETS_REQUIRED=true`, which `release.yml` sets on `main`, it fails instead. The first consumer wires `CI_SECRETS_REQUIRED` in the same PR.

Violation:

```yaml
jobs:
  test:
    env:
      STRIPE_TEST_KEY: ${{ secrets.STRIPE_TEST_KEY }}
    steps:
      - run: make test
```

Fix:

```yaml
- name: Test the payments client
  env:
    STRIPE_TEST_KEY: ${{ secrets.STRIPE_TEST_KEY }}
  run: make test
```

and the test calls `t.Skip` when `STRIPE_TEST_KEY` is empty, unless `CI_SECRETS_REQUIRED` is `true`.

Source: `boundary-discipline`.

## Deliver `prd` secrets through the environment only

- A production process reads its secrets from its environment only: no `doppler run`, no Doppler CLI, no secrets file.
- Delivery is a Doppler native sync to the runtime platform; a read-only `prd` service token stored only in that platform is the fallback.
- A `prd` value, or a token that reads `prd`, never goes into a GitHub repository secret, only into a protected GitHub Environment that only the deploy job declares.
- The `prd` sync or token is declared in the `prd` workspace of `iac/environments/` and applied locally, never in CI.

Violation:

```yaml
- run: doppler run --config prd -- ./deploy.sh
  env:
    DOPPLER_TOKEN: ${{ secrets.DOPPLER_PRD_TOKEN }}
```

Fix: declare the `prd` sync to the platform in `iac/environments/` and let the deployed process read its environment.

Source: `boundary-discipline`.
