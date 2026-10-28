-include ../../.env
-include .env
export

HEALTH_PORT ?= 8092

DB_NAME := audit
PSQL := docker compose -f ../../docker-compose.yml exec -T postgres psql -q -v ON_ERROR_STOP=1 -U postgres -d postgres
KAFKA_TOPICS := docker compose -f ../../docker-compose.yml exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:19092

DEBUG_GCFLAGS := all=-N -l

.PHONY: build.consumer
build.consumer:
	@go build -gcflags="$(DEBUG_GCFLAGS)" -o ./bin/audit_consumer ./cmd/consumer

.PHONY: run.consumer
run.consumer:
	@OTEL_SERVICE_NAME=audit-consumer HEALTH_PORT=$(HEALTH_PORT) ./bin/audit_consumer

.PHONY: db.setup
db.setup:
	@printf '%s\n' \
		"SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', '$(DB_NAME)', '$(DB_NAME)') WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$(DB_NAME)')\gexec" \
		"SELECT format('CREATE DATABASE %I OWNER %I', '$(DB_NAME)', '$(DB_NAME)') WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$(DB_NAME)')\gexec" \
		| $(PSQL)

.PHONY: db.migrate
db.migrate:
	@go build -o ./bin/audit_migrate ./cmd/migrate
	@OTEL_SERVICE_NAME=audit-migrate ./bin/audit_migrate

.PHONY: db.rollback
db.rollback:
	@go build -o ./bin/audit_migrate ./cmd/migrate
	@OTEL_SERVICE_NAME=audit-migrate ./bin/audit_migrate down

.PHONY: db.reset
db.reset:
	@echo "DROP DATABASE IF EXISTS $(DB_NAME) WITH (FORCE)" | $(PSQL)
	@$(MAKE) -s db.setup db.migrate

.PHONY: kafka.topics
kafka.topics:
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.user.v1.audit.dlq --partitions 1 --replication-factor 1 --config retention.ms=-1
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.session.v1.audit.dlq --partitions 1 --replication-factor 1 --config retention.ms=-1
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.organization.v1.audit.dlq --partitions 1 --replication-factor 1 --config retention.ms=-1
	@$(KAFKA_TOPICS) --create --if-not-exists --topic iam.invitation.v1.audit.dlq --partitions 1 --replication-factor 1 --config retention.ms=-1
# example:begin
	@$(KAFKA_TOPICS) --create --if-not-exists --topic notes.note.v1.audit.dlq --partitions 1 --replication-factor 1 --config retention.ms=-1
# example:end
