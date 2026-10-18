package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
)

func TestDatabaseURL_Set(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/notes")

	url, err := config.DatabaseURL()

	require.NoError(t, err)
	require.Equal(t, "postgres://user:pass@db:5432/notes", url)
}

func TestDatabaseURL_MissingSaysWhichKeyToAdd(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := config.DatabaseURL()

	require.ErrorIs(t, err, config.ErrDatabaseURLMissing)
	require.Equal(t,
		"DATABASE_URL is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/notes/.env.example to backend/services/notes/.env",
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
			"backend/services/notes/.env.example to backend/services/notes/.env",
		err.Error(),
	)
}

func TestHealthPort_Set(t *testing.T) {
	t.Setenv("HEALTH_PORT", "8091")

	port, err := config.HealthPort()

	require.NoError(t, err)
	require.Equal(t, 8091, port)
}

func TestHealthPort_Invalid(t *testing.T) {
	for _, value := range []string{"", "http", "0", "70000"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("HEALTH_PORT", value)

			_, err := config.HealthPort()

			require.ErrorIs(t, err, config.ErrInvalidHealthPort)
			require.Equal(t,
				"HEALTH_PORT must be a TCP port: start the worker with its make run.* target, "+
					"which sets it from the service's .env (e.g. FORWARDER_HEALTH_PORT)",
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
			"backend/services/notes/.env.example to backend/services/notes/.env",
		err.Error(),
	)
}
