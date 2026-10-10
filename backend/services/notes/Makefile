-include ../../.env
-include .env
export

FRONTEND_PORT ?= 8080

# Debug build (no optimizations, no inlining) so the VS Code "Attach" config can set breakpoints.
.PHONY: build.api.frontend
build.api.frontend:
	@go build -gcflags="all=-N -l" -o ./bin/notes_api_frontend ./cmd/api/frontend

.PHONY: run.api.frontend
run.api.frontend:
	@OTEL_SERVICE_NAME=notes-api-frontend PORT=$(FRONTEND_PORT) ./bin/notes_api_frontend
