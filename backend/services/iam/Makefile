-include ../../.env
-include .env
export

WORKER_HEALTH_PORT ?= 8093

DB_NAME := iam
PSQL := docker compose -f ../../docker-compose.yml exec -T postgres psql -q -v ON_ERROR_STOP=1 -U postgres -d postgres
KAFKA_TOPICS := docker compose -f ../../docker-compose.yml exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:19092

DEBUG_GCFLAGS := all=-N -l

.PHONY: build.worker
build.worker:
	@go build -gcflags="$(DEBUG_GCFLAGS)" -o ./bin/iam_worker ./cmd/worker

.PHONY: run.worker
run.worker:
	@OTEL_SERVICE_NAME=iam-worker HEALTH_PORT=$(WORKER_HEALTH_PORT) ./bin/iam_worker

.PHONY: db.setup
db.setup:
	@printf '%s\n' \
		"SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', '$(DB_NAME)', '$(DB_NAME)') WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$(DB_NAME)')\gexec" \
		"SELECT format('CREATE DATABASE %I OWNER %I', '$(DB_NAME)', '$(DB_NAME)') WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$(DB_NAME)')\gexec" \
		| $(PSQL)

.PHONY: db.migrate
db.migrate:
	@go build -o ./bin/iam_migrate ./cmd/migrate
	@OTEL_SERVICE_NAME=iam-migrate ./bin/iam_migrate

.PHONY: db.rollback
db.rollback:
	@go build -o ./bin/iam_migrate ./cmd/migrate
	@OTEL_SERVICE_NAME=iam-migrate ./bin/iam_migrate down

.PHONY: db.reset
db.reset:
	@echo "DROP DATABASE IF EXISTS $(DB_NAME) WITH (FORCE)" | $(PSQL)
	@$(MAKE) -s db.setup db.migrate

.PHONY: kafka.topics
kafka.topics:
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.user.v1 --partitions 3 --replication-factor 1
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.session.v1 --partitions 3 --replication-factor 1
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.organization.v1 --partitions 3 --replication-factor 1
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.invitation.v1 --partitions 3 --replication-factor 1

.PHONY: schemas.register
schemas.register:
	@go build -o ./bin/iam_schemas ./cmd/schemas
	@OTEL_SERVICE_NAME=iam-schemas ./bin/iam_schemas
