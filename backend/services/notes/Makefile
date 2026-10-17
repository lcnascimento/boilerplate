-include ../../.env
-include .env
export

FRONTEND_PORT ?= 8080

# psql runs inside the container: the host needs Docker only.
DB_NAME := notes
PSQL := docker exec -i postgres psql -q -v ON_ERROR_STOP=1 -U postgres -d postgres

# Debug build (no optimizations, no inlining) so the VS Code "Attach" config can set breakpoints.
.PHONY: build.api.frontend
build.api.frontend:
	@go build -gcflags="all=-N -l" -o ./bin/notes_api_frontend ./cmd/api/frontend

.PHONY: run.api.frontend
run.api.frontend:
	@OTEL_SERVICE_NAME=notes-api-frontend PORT=$(FRONTEND_PORT) ./bin/notes_api_frontend

.PHONY: db.setup
db.setup:
	@printf '%s\n' \
		"SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', '$(DB_NAME)', '$(DB_NAME)') WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$(DB_NAME)')\gexec" \
		"SELECT format('CREATE DATABASE %I OWNER %I', '$(DB_NAME)', '$(DB_NAME)') WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$(DB_NAME)')\gexec" \
		| $(PSQL)

.PHONY: db.migrate
db.migrate:
	@go build -o ./bin/notes_migrate ./cmd/migrate
	@OTEL_SERVICE_NAME=notes-migrate ./bin/notes_migrate

.PHONY: db.rollback
db.rollback:
	@go build -o ./bin/notes_migrate ./cmd/migrate
	@OTEL_SERVICE_NAME=notes-migrate ./bin/notes_migrate down

.PHONY: db.reset
db.reset:
	@echo "DROP DATABASE IF EXISTS $(DB_NAME) WITH (FORCE)" | $(PSQL)
	@$(MAKE) -s db.setup db.migrate
