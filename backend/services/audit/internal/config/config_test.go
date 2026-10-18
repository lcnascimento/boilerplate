package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/config"
)

func TestDatabaseURL_Set(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/audit")

	url, err := config.DatabaseURL()

	require.NoError(t, err)
	require.Equal(t, "postgres://user:pass@db:5432/audit", url)
}

func TestDatabaseURL_MissingSaysWhichKeyToAdd(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := config.DatabaseURL()

	require.ErrorIs(t, err, config.ErrDatabaseURLMissing)
	require.Equal(t,
		"DATABASE_URL is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/audit/.env.example to backend/services/audit/.env",
		err.Error(),
	)
}

func TestKafkaBrokers_SplitsTheList(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "kafka-1:9092, kafka-2:9092")

	brokers, err := config.KafkaBrokers()

	require.NoError(t, err)
	require.Equal(t, []string{"kafka-1:9092", "kafka-2:9092"}, brokers)
}

func TestKafkaBrokers_MissingSaysWhichKeyToAdd(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")

	_, err := config.KafkaBrokers()

	require.ErrorIs(t, err, config.ErrKafkaBrokersMissing)
	require.Contains(t, err.Error(), "KAFKA_BROKERS is not set: run `make -C backend env`")
}

func TestHealthPort_Set(t *testing.T) {
	t.Setenv("HEALTH_PORT", "8092")

	port, err := config.HealthPort()

	require.NoError(t, err)
	require.Equal(t, 8092, port)
}

func TestHealthPort_Invalid(t *testing.T) {
	cases := map[string]string{
		"missing":      "",
		"not a number": "eighty",
		"out of range": "70000",
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("HEALTH_PORT", value)

			_, err := config.HealthPort()

			require.ErrorIs(t, err, config.ErrHealthPortInvalid)
		})
	}
}
