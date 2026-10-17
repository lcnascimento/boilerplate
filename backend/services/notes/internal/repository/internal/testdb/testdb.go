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

// DefaultAdminURL is for tests only: the service's own role never needs CREATEDB.
const DefaultAdminURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" //nolint:gosec // local docker-compose credentials

const pingTimeout = 5 * time.Second

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

	if err := ping(t.Context(), conf); err != nil {
		t.Fatalf("postgres is not reachable at POSTGRES_ADMIN_URL (%s:%s): %v\n"+
			"Start it with `tilt up` or `docker compose up -d postgres` in backend/, "+
			"or run `make test.unit` to skip the component tests.", conf.Host, conf.Port, err)
	}
}

func ping(ctx context.Context, conf *pgtestdb.Config) error {
	db, err := conf.Connect()
	if err != nil {
		return err
	}

	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	return db.PingContext(ctx)
}

type migrator struct{}

func (migrator) Hash() (string, error) {
	return common.HashDirs(migrations.FS, "*.sql", ".")
}

//nolint:gocritic // hugeParam: the signature is pgtestdb.Migrator's.
func (migrator) Migrate(ctx context.Context, db *sql.DB, _ pgtestdb.Config) error {
	return migrations.Up(ctx, db)
}
