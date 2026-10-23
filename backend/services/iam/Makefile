-include ../../.env
-include .env
export

DB_NAME := iam
PSQL := docker exec -i postgres psql -q -v ON_ERROR_STOP=1 -U postgres -d postgres
KAFKA_TOPICS := docker exec kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:19092

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
