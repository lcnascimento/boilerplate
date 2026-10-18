package config

import (
	"strings"

	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
)

const maxPort = 65535

var (
	ErrDatabaseURLMissing       = errMissing("DATABASE_URL").WithCode("DATABASE_URL_MISSING")
	ErrSchemaRegistryURLMissing = errMissing("SCHEMA_REGISTRY_URL").WithCode("SCHEMA_REGISTRY_URL_MISSING")
	ErrKafkaBrokersMissing      = errMissing("KAFKA_BROKERS").WithCode("KAFKA_BROKERS_MISSING")

	ErrInvalidHealthPort = errors.New(
		"HEALTH_PORT must be a TCP port: start the worker with its make run.* target, " +
			"which sets it from the service's .env (e.g. FORWARDER_HEALTH_PORT)",
	).WithCode("INVALID_HEALTH_PORT")
)

func errMissing(key string) errors.CustomError {
	return errors.New(
		"%s is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/notes/.env.example to backend/services/notes/.env",
		key,
	)
}

func DatabaseURL() (string, error) {
	return required("DATABASE_URL", ErrDatabaseURLMissing)
}

func SchemaRegistryURL() (string, error) {
	return required("SCHEMA_REGISTRY_URL", ErrSchemaRegistryURLMissing)
}

func KafkaBrokers() ([]string, error) {
	var brokers []string

	for _, broker := range env.GetList[string]("KAFKA_BROKERS") {
		if broker = strings.TrimSpace(broker); broker != "" {
			brokers = append(brokers, broker)
		}
	}

	if len(brokers) == 0 {
		return nil, ErrKafkaBrokersMissing
	}

	return brokers, nil
}

func HealthPort() (int, error) {
	port := env.Get[int]("HEALTH_PORT")
	if port <= 0 || port > maxPort {
		return 0, ErrInvalidHealthPort
	}

	return port, nil
}

func required(key string, missing error) (string, error) {
	value := env.Get[string](key)
	if value == "" {
		return "", missing
	}

	return value, nil
}
