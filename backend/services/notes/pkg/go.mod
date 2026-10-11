module github.com/lcnascimento/boilerplate/backend/services/notes/pkg

go 1.27.2

// go-kit modules reference each other through local replaces, which leak
// zero pseudo-versions into their requirements: exclude them so tags win.
exclude (
	github.com/lcnascimento/go-kit/env v0.0.0-00010101000000-000000000000
	github.com/lcnascimento/go-kit/errors v0.0.0-00010101000000-000000000000
	github.com/lcnascimento/go-kit/http v0.0.0-00010101000000-000000000000
	github.com/lcnascimento/go-kit/o11y v0.0.0-00010101000000-000000000000
	github.com/lcnascimento/go-kit/util v0.0.0-00010101000000-000000000000
	github.com/lcnascimento/go-kit/validator v0.0.0-00010101000000-000000000000
)

require (
	github.com/lcnascimento/go-kit/errors v1.9.0
	github.com/lcnascimento/go-kit/util v1.4.3
	github.com/lcnascimento/go-kit/validator v1.10.1
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.23.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/lcnascimento/go-kit/o11y v1.6.3 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.39.0 // indirect
)
