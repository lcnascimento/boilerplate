package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/postgres/migrate"
	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/migrations"
)

const (
	checkViolation     = "23514"
	exclusionViolation = "23P01"
)

var wantTables = []string{
	"account",
	"identity_change",
	"invitation",
	"jwks",
	"member",
	"organization",
	"rateLimit",
	"session",
	"user",
	"verification",
	"watermill_iam_outbox",
	"watermill_offsets_iam_outbox",
}

func TestComponentMigrations_CreatesTheBetterAuthAndAuditTables(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	require.Equal(t, wantTables, publicTables(t, db))
}

func TestComponentMigrations_AcceptsOnlyTheKnownRoles(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)
	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO "user" (id, name, email, "emailVerified")
		VALUES ('00000000-0000-4000-8000-000000000001', 'Ada', 'ada@acme.test', false);
		INSERT INTO organization (id, name, slug, "createdAt")
		VALUES ('00000000-0000-4000-8000-000000000002', 'Acme', 'acme', now())`)
	require.NoError(t, err)

	for _, role := range []string{"owner", "admin", "viewer"} {
		require.NoError(t, insertMember(t, db, role), role)
		require.NoError(t, insertInvitation(t, db, role), role)
	}

	for _, role := range []string{"member", "owner,admin", "Owner", ""} {
		requireCheckViolation(t, insertMember(t, db, role), role)
		requireCheckViolation(t, insertInvitation(t, db, role), role)
	}
}

func TestComponentMigrations_AllowsOneOwnerPerOrganization(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)
	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO "user" (id, name, email, "emailVerified") VALUES
			('00000000-0000-4000-8000-000000000001', 'Ada', 'ada@acme.test', false),
			('00000000-0000-4000-8000-000000000003', 'Grace', 'grace@acme.test', false);
		INSERT INTO organization (id, name, slug, "createdAt") VALUES
			('00000000-0000-4000-8000-000000000002', 'Acme', 'acme', now()),
			('00000000-0000-4000-8000-000000000004', 'Globex', 'globex', now());
		INSERT INTO member (id, "organizationId", "userId", role, "createdAt") VALUES
			('00000000-0000-4000-8000-000000000011', '00000000-0000-4000-8000-000000000002',
				'00000000-0000-4000-8000-000000000001', 'owner', now()),
			('00000000-0000-4000-8000-000000000012', '00000000-0000-4000-8000-000000000002',
				'00000000-0000-4000-8000-000000000003', 'admin', now()),
			('00000000-0000-4000-8000-000000000013', '00000000-0000-4000-8000-000000000004',
				'00000000-0000-4000-8000-000000000003', 'owner', now())`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`UPDATE member SET role = 'owner' WHERE id = '00000000-0000-4000-8000-000000000012'`)
	requireExclusionViolation(t, err)

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(t.Context(),
		`UPDATE member SET role = 'owner' WHERE id = '00000000-0000-4000-8000-000000000012'`)
	require.NoError(t, err)
	_, err = tx.ExecContext(t.Context(),
		`UPDATE member SET role = 'admin' WHERE id = '00000000-0000-4000-8000-000000000011'`)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	var owner string
	require.NoError(t, db.QueryRowContext(t.Context(), `
		SELECT "userId" FROM member
		WHERE "organizationId" = '00000000-0000-4000-8000-000000000002' AND role = 'owner'`).Scan(&owner))
	require.Equal(t, "00000000-0000-4000-8000-000000000003", owner)
}

func TestComponentMigrations_EveryMigrationRollsBackAndReapplies(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	for rollbacks := 0; ; rollbacks++ {
		require.Less(t, rollbacks, 1000, "Down never ran out of migrations")

		err := migrate.Down(t.Context(), db, migrations.FS)
		if errors.Is(err, migrate.ErrNothingToRollBack) {
			break
		}

		require.NoError(t, err)
	}

	require.Empty(t, publicTables(t, db))
	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))
	require.Equal(t, wantTables, publicTables(t, db))
}

func insertMember(t *testing.T, db *sql.DB, role string) error {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO member ("organizationId", "userId", role, "createdAt")
		VALUES ('00000000-0000-4000-8000-000000000002', '00000000-0000-4000-8000-000000000001', $1, now())`,
		role)

	return err
}

func insertInvitation(t *testing.T, db *sql.DB, role string) error {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO invitation ("organizationId", email, role, status, "expiresAt", "inviterId")
		VALUES ('00000000-0000-4000-8000-000000000002', 'grace@acme.test', $1, 'pending', now(),
			'00000000-0000-4000-8000-000000000001')`,
		role)

	return err
}

func requireCheckViolation(t *testing.T, err error, role string) {
	t.Helper()

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr, role)
	require.Equal(t, checkViolation, pgErr.Code, role)
}

func requireExclusionViolation(t *testing.T, err error) {
	t.Helper()

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, exclusionViolation, pgErr.Code)
	require.Equal(t, "member_one_owner_per_organization", pgErr.ConstraintName)
}

func publicTables(t *testing.T, db *sql.DB) []string {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'goose_db_version'
		ORDER BY table_name COLLATE "C"`)
	require.NoError(t, err)

	defer func() { require.NoError(t, rows.Close()) }()

	var tables []string

	for rows.Next() {
		var table string
		require.NoError(t, rows.Scan(&table))

		tables = append(tables, table)
	}

	require.NoError(t, rows.Err())

	return tables
}

func newDatabase(t *testing.T) *sql.DB {
	t.Helper()

	if testing.Short() {
		t.Skip("component test")
	}

	return testdb.NewEmpty(t)
}
