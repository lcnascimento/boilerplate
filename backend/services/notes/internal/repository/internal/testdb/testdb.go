// Package testdb gives each component test of the repository layer its own
// Postgres database, cloned from a template that pgtestdb keeps across runs
// and rebuilds only when the migrations change. A failing test's database is
// kept, and its URL is in the test log.
package testdb

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"

	// Registers the "pgx" database/sql driver pgtestdb connects with.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/lcnascimento/go-kit/env"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
)

// DefaultAdminURL is the superuser of backend/docker-compose.yml. Tests only:
// the service's own role never needs CREATEDB.
const DefaultAdminURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" //nolint:gosec // local docker-compose credentials

const pingTimeout = 5 * time.Second

// New returns a pool on a fresh database with every migration applied. It
// fails the test, never skips it, when Postgres is not reachable.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	instance := pgtestdb.Custom(t, reachableAdminConfig(t), migrator{})

	pool, err := pgxpool.New(t.Context(), instance.URL())
	if err != nil {
		t.Fatalf("testdb: connect to %s: %v", instance.Database, err)
	}

	t.Cleanup(pool.Close)

	return pool
}

// NewEmpty returns a database with no migration applied, for the tests of the
// migrations themselves. pgtestdb closes it at cleanup.
func NewEmpty(t *testing.T) *sql.DB {
	t.Helper()

	return pgtestdb.New(t, reachableAdminConfig(t), pgtestdb.NoopMigrator{})
}

func reachableAdminConfig(t *testing.T) pgtestdb.Config {
	t.Helper()

	conf := adminConfig(t)
	requireReachable(t, &conf)

	return conf
}

func adminConfig(t *testing.T) pgtestdb.Config {
	t.Helper()

	raw := env.Get("POSTGRES_ADMIN_URL", env.WithDefaultValue(DefaultAdminURL))

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("testdb: POSTGRES_ADMIN_URL is not a URL: %v", err)
	}

	password, _ := u.User.Password()

	port := u.Port()
	if port == "" {
		port = "5432"
	}

	return pgtestdb.Config{
		DriverName: "pgx",
		Host:       u.Hostname(),
		Port:       port,
		User:       u.User.Username(),
		Password:   password,
		Database:   strings.TrimPrefix(u.Path, "/"),
		Options:    u.RawQuery,
	}
}

func requireReachable(t *testing.T, conf *pgtestdb.Config) {
	t.Helper()

	db, err := conf.Connect()
	if err == nil {
		defer func() { _ = db.Close() }()

		ctx, cancel := context.WithTimeout(t.Context(), pingTimeout)
		defer cancel()

		err = db.PingContext(ctx)
	}

	if err != nil {
		t.Fatalf("postgres is not reachable at POSTGRES_ADMIN_URL (%s:%s): %v\n"+
			"Start it with `tilt up` or `docker compose up -d postgres` in backend/, "+
			"or run `make test.unit` to skip the component tests.", conf.Host, conf.Port, err)
	}
}

// migrator builds pgtestdb's template with the service's own migrations.Up.
type migrator struct{}

func (migrator) Hash() (string, error) {
	return common.HashDirs(migrations.FS, "*.sql", ".")
}

//nolint:gocritic // hugeParam: the signature is pgtestdb.Migrator's.
func (migrator) Migrate(ctx context.Context, db *sql.DB, _ pgtestdb.Config) error {
	return migrations.Up(ctx, db)
}
