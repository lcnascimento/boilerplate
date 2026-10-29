package config

import (
	"strings"

	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
)

const maxPort = 65535

var (
	ErrDatabaseURLMissing       = errMissing("IAM_DATABASE_URL").WithCode("IAM_DATABASE_URL_MISSING")
	ErrSchemaRegistryURLMissing = errMissing("SCHEMA_REGISTRY_URL").WithCode("SCHEMA_REGISTRY_URL_MISSING")
	ErrKafkaBrokersMissing      = errMissing("KAFKA_BROKERS").WithCode("KAFKA_BROKERS_MISSING")

	ErrServiceNameMissing = errors.New(
		"OTEL_SERVICE_NAME is not set: start the process with its make run.* target, " +
			"which sets it to the process's service.name",
	).WithCode("SERVICE_NAME_MISSING")

	ErrInvalidHealthPort = errors.New(
		"HEALTH_PORT must be a TCP port: start the worker with its make run.* target, " +
			"which sets it from the service's .env (e.g. WORKER_HEALTH_PORT)",
	).WithCode("INVALID_HEALTH_PORT")
)

func errMissing(key string) errors.CustomError {
	return errors.New(
		"%s is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/iam/.env.example to backend/services/iam/.env",
		key,
	)
}

func DatabaseURL() (string, error) {
	return required("IAM_DATABASE_URL", ErrDatabaseURLMissing)
}

func ServiceName() (string, error) {
	return required("OTEL_SERVICE_NAME", ErrServiceNameMissing)
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
