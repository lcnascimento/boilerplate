package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/config"
)

func TestDatabaseURL_Set(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/iam")

	url, err := config.DatabaseURL()

	require.NoError(t, err)
	require.Equal(t, "postgres://user:pass@db:5432/iam", url)
}

func TestDatabaseURL_MissingSaysWhichKeyToAdd(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := config.DatabaseURL()

	require.ErrorIs(t, err, config.ErrDatabaseURLMissing)
	require.Equal(t,
		"DATABASE_URL is not set: run `make -C backend env` to add the missing keys of "+
			"backend/services/iam/.env.example to backend/services/iam/.env",
		err.Error(),
	)
}
