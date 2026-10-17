-include ../../.env
-include .env
export

FRONTEND_PORT ?= 8080
FORWARDER_HEALTH_PORT ?= 8091
SCHEMA_REGISTRY_URL ?= http://localhost:8081

DB_NAME := notes
PSQL := docker exec -i postgres psql -q -v ON_ERROR_STOP=1 -U postgres -d postgres
KAFKA_TOPICS := docker exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:19092

DEBUG_GCFLAGS := all=-N -l

.PHONY: build.api.frontend
build.api.frontend:
	@go build -gcflags="$(DEBUG_GCFLAGS)" -o ./bin/notes_api_frontend ./cmd/api/frontend

.PHONY: run.api.frontend
run.api.frontend:
	@OTEL_SERVICE_NAME=notes-api-frontend PORT=$(FRONTEND_PORT) ./bin/notes_api_frontend

.PHONY: build.worker.forwarder
build.worker.forwarder:
	@go build -gcflags="$(DEBUG_GCFLAGS)" -o ./bin/notes_worker_forwarder ./cmd/worker/forwarder

.PHONY: run.worker.forwarder
run.worker.forwarder:
	@OTEL_SERVICE_NAME=notes-worker-forwarder HEALTH_PORT=$(FORWARDER_HEALTH_PORT) ./bin/notes_worker_forwarder

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

.PHONY: kafka.topics
kafka.topics:
	@$(KAFKA_TOPICS) --create --if-not-exists --topic notes.note.v1 --partitions 3 --replication-factor 1

.PHONY: schemas.register
schemas.register:
	@go build -o ./bin/notes_schemas ./cmd/schemas
	@OTEL_SERVICE_NAME=notes-schemas ./bin/notes_schemas
