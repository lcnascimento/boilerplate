module github.com/lcnascimento/boilerplate/backend/services/notes

go 1.27.2

replace github.com/lcnascimento/boilerplate/backend/services/notes/pkg => ./pkg

require (
	github.com/gorilla/mux v1.8.1
	github.com/lcnascimento/boilerplate/backend/services/notes/pkg v0.0.0-00010101000000-000000000000
	github.com/lcnascimento/go-kit/errors v1.9.3
	github.com/lcnascimento/go-kit/http v1.4.8
	github.com/lcnascimento/go-kit/o11y v1.6.6
	github.com/lcnascimento/go-kit/validator v1.11.0
	github.com/stretchr/testify v1.12.1
	go.opentelemetry.io/otel v1.45.0
	go.opentelemetry.io/otel/trace v1.45.0
	go.uber.org/fx v1.24.0
	go.uber.org/mock v0.6.0
)

require (
	github.com/caarlos0/env/v10 v10.0.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.23.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/lcnascimento/go-kit/env v1.2.2 // indirect
	github.com/lcnascimento/go-kit/util v1.4.5 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelslog v0.20.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.70.0 // indirect
	go.opentelemetry.io/contrib/processors/minsev v0.16.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.21.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.21.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.45.0 // indirect
	go.opentelemetry.io/otel/log v0.21.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.21.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260803160001-6ac0973c030d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260803160001-6ac0973c030d // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
