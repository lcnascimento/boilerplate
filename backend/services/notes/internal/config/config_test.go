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
		"DATABASE_URL is not set: add the DATABASE_URL line of backend/services/notes/.env.example "+
			"to backend/services/notes/.env (`make env` creates .env only when it is missing)",
		err.Error(),
	)
}
