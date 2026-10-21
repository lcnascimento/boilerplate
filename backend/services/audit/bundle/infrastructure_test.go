package bundle_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	"github.com/lcnascimento/go-kit/env"

	"github.com/lcnascimento/boilerplate/backend/services/audit/bundle"
)

const defaultAdminURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

func TestComponentNewPool_ConnectionsCarryTheServiceName(t *testing.T) {
	if testing.Short() {
		t.Skip("component test: needs Postgres")
	}

	t.Setenv("DATABASE_URL", env.Get("POSTGRES_ADMIN_URL", env.WithDefaultValue(defaultAdminURL)))
	t.Setenv("OTEL_SERVICE_NAME", "audit-consumer")

	lifecycle := fxtest.NewLifecycle(t)

	pool, err := bundle.NewPool(lifecycle)
	require.NoError(t, err)

	lifecycle.RequireStart()
	t.Cleanup(lifecycle.RequireStop)

	var name string

	require.NoError(t, pool.QueryRow(t.Context(),
		"SELECT application_name FROM pg_stat_activity WHERE pid = pg_backend_pid()",
	).Scan(&name))
	require.Equal(t, "audit-consumer", name)
}
