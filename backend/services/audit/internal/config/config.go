package config

import (
	"strconv"
	"strings"

	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
)

const maxPort = 65535

var (
	ErrDatabaseURLMissing  = errMissing("DATABASE_URL").WithCode("DATABASE_URL_MISSING")
	ErrKafkaBrokersMissing = errMissing("KAFKA_BROKERS").WithCode("KAFKA_BROKERS_MISSING")
	ErrHealthPortInvalid   = errors.New(
		"HEALTH_PORT must be a TCP port: copy its line from backend/services/audit/.env.example to backend/services/audit/.env",
	).WithCode("HEALTH_PORT_INVALID")
	ErrServiceNameMissing = errors.New(
		"OTEL_SERVICE_NAME is not set: start the process with its make run.* target, " +
			"which sets it to the process's service.name",
	).WithCode("SERVICE_NAME_MISSING")
)

func errMissing(key string) errors.CustomError {
	return errors.New(
		"%s is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/audit/.env.example to backend/services/audit/.env",
		key,
	)
}

func DatabaseURL() (string, error) {
	return required("DATABASE_URL", ErrDatabaseURLMissing)
}

func ServiceName() (string, error) {
	return required("OTEL_SERVICE_NAME", ErrServiceNameMissing)
}

func KafkaBrokers() ([]string, error) {
	raw, err := required("KAFKA_BROKERS", ErrKafkaBrokersMissing)
	if err != nil {
		return nil, err
	}

	brokers := strings.Split(raw, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	return brokers, nil
}

func HealthPort() (int, error) {
	raw := env.Get[string]("HEALTH_PORT")

	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, ErrHealthPortInvalid.WithAttribute("value", raw).WithCause(err)
	}

	if port < 1 || port > maxPort {
		return 0, ErrHealthPortInvalid.WithAttribute("value", raw)
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
