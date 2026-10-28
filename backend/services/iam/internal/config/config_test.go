package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/config"
)

func TestDatabaseURL_Set(t *testing.T) {
	t.Setenv("IAM_DATABASE_URL", "postgres://user:pass@db:5432/iam")

	url, err := config.DatabaseURL()

	require.NoError(t, err)
	require.Equal(t, "postgres://user:pass@db:5432/iam", url)
}

func TestDatabaseURL_MissingSaysWhichKeyToAdd(t *testing.T) {
	t.Setenv("IAM_DATABASE_URL", "")

	_, err := config.DatabaseURL()

	require.ErrorIs(t, err, config.ErrDatabaseURLMissing)
	require.Equal(t,
		"IAM_DATABASE_URL is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/iam/.env.example to backend/services/iam/.env",
		err.Error(),
	)
}

func TestKafkaBrokers_Set(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "kafka-1:9092, kafka-2:9092")

	brokers, err := config.KafkaBrokers()

	require.NoError(t, err)
	require.Equal(t, []string{"kafka-1:9092", "kafka-2:9092"}, brokers)
}

func TestKafkaBrokers_MissingSaysWhichKeyToAdd(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", " , ")

	_, err := config.KafkaBrokers()

	require.ErrorIs(t, err, config.ErrKafkaBrokersMissing)
	require.Equal(t,
		"KAFKA_BROKERS is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/iam/.env.example to backend/services/iam/.env",
		err.Error(),
	)
}

func TestHealthPort_Set(t *testing.T) {
	t.Setenv("HEALTH_PORT", "8093")

	port, err := config.HealthPort()

	require.NoError(t, err)
	require.Equal(t, 8093, port)
}

func TestHealthPort_Invalid(t *testing.T) {
	for _, value := range []string{"", "http", "0", "70000"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("HEALTH_PORT", value)

			_, err := config.HealthPort()

			require.ErrorIs(t, err, config.ErrInvalidHealthPort)
			require.Equal(t,
				"HEALTH_PORT must be a TCP port: start the worker with its make run.* target, "+
					"which sets it from the service's .env (e.g. WORKER_HEALTH_PORT)",
				err.Error(),
			)
		})
	}
}

func TestSchemaRegistryURL_Set(t *testing.T) {
	t.Setenv("SCHEMA_REGISTRY_URL", "http://registry:8081")

	url, err := config.SchemaRegistryURL()

	require.NoError(t, err)
	require.Equal(t, "http://registry:8081", url)
}

func TestSchemaRegistryURL_MissingSaysWhichKeyToAdd(t *testing.T) {
	t.Setenv("SCHEMA_REGISTRY_URL", "")

	_, err := config.SchemaRegistryURL()

	require.ErrorIs(t, err, config.ErrSchemaRegistryURLMissing)
	require.Equal(t,
		"SCHEMA_REGISTRY_URL is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/iam/.env.example to backend/services/iam/.env",
		err.Error(),
	)
}

func TestServiceName_Set(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "iam-worker")

	name, err := config.ServiceName()

	require.NoError(t, err)
	require.Equal(t, "iam-worker", name)
}

func TestServiceName_MissingSaysHowToSetIt(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")

	_, err := config.ServiceName()

	require.ErrorIs(t, err, config.ErrServiceNameMissing)
	require.Equal(t,
		"OTEL_SERVICE_NAME is not set: start the process with its make run.* target, "+
			"which sets it to the process's service.name",
		err.Error(),
	)
}
