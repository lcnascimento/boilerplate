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

The code that handles an error logs it once: the application, or the port when the error originates there. Code that only passes an error on records it on the span and in the metrics (`error.type`) and does not log it, so one failure is one log line and one tracked exception.

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
