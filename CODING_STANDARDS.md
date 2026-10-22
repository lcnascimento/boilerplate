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

Fix: delete both lines. Encode the postgres requirement as a target prerequisite; if humans need the usage notes, write them as a document with `foundation:documentation`.

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
