package notes_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
)

var (
	acme   = iamdomain.MustParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")
	globex = iamdomain.MustParseOrganizationID("org_4d7a1c3e-5b6f-4a8b-9c0d-1e2f3a4b5c6d")

	acmeOwner   = actortest.Owner(acme)
	globexOwner = actortest.Owner(globex)
)

func TestComponentRepository_StartsEmpty(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)

	got, err := repo.List(t.Context(), acme)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestComponentRepository_CreateThenGet(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 123456000, time.UTC))

	require.NoError(t, repo.Create(t.Context(), acmeOwner, note))

	got, err := repo.Get(t.Context(), note.OrganizationID, note.ID)

	require.NoError(t, err)
	require.Equal(t, note, got)
}

func TestComponentRepository_GetUnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)

	_, err := repo.Get(t.Context(), acme, domain.NewNoteID())

	require.ErrorIs(t, err, domain.ErrNoteNotFound)
}

func TestComponentRepository_ListReturnsNewestFirst(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	base := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	oldest := newNote("oldest", base)
	newest := newNote("newest", base.Add(2*time.Hour))
	middle := newNote("middle", base.Add(time.Hour))

	for _, note := range []*domain.Note{oldest, newest, middle} {
		require.NoError(t, repo.Create(t.Context(), acmeOwner, note))
	}

	got, err := repo.List(t.Context(), acme)

	require.NoError(t, err)
	require.Equal(t, []string{"newest", "middle", "oldest"}, titles(got))
}

func TestComponentRepository_ListBreaksTimestampTiesByID(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	at := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	first := newNote("first", at)
	second := newNote("second", at)

	require.NoError(t, repo.Create(t.Context(), acmeOwner, first))
	require.NoError(t, repo.Create(t.Context(), acmeOwner, second))

	got, err := repo.List(t.Context(), acme)

	require.NoError(t, err)
	require.Equal(t, []string{"second", "first"}, titles(got))
}

func TestComponentRepository_DeleteRemovesNote(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	require.NoError(t, repo.Create(t.Context(), acmeOwner, note))

	require.NoError(t, repo.Delete(t.Context(), acmeOwner, note.ID))

	_, err := repo.Get(t.Context(), note.OrganizationID, note.ID)
	require.ErrorIs(t, err, domain.ErrNoteNotFound)
}

func TestComponentRepository_DeleteUnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)

	err := repo.Delete(t.Context(), acmeOwner, domain.NewNoteID())

	require.ErrorIs(t, err, domain.ErrNoteNotFound)
}

func TestComponentRepository_ListReturnsOnlyTheOrganizationsNotes(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	at := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	require.NoError(t, repo.Create(t.Context(), acmeOwner, newNote("da Acme", at)))
	require.NoError(t, repo.Create(t.Context(), globexOwner, newGlobexNote(at)))

	acmeNotes, err := repo.List(t.Context(), acme)
	require.NoError(t, err)

	globexNotes, err := repo.List(t.Context(), globex)
	require.NoError(t, err)

	require.Equal(t, []string{"da Acme"}, titles(acmeNotes))
	require.Equal(t, []string{"da Globex"}, titles(globexNotes))
}

func TestComponentRepository_GetFromAnotherOrganizationReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	note := newGlobexNote(time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	require.NoError(t, repo.Create(t.Context(), globexOwner, note))

	_, err := repo.Get(t.Context(), acme, note.ID)

	require.ErrorIs(t, err, domain.ErrNoteNotFound)
}

func TestComponentRepository_DeleteFromAnotherOrganizationKeepsTheNote(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	note := newGlobexNote(time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	require.NoError(t, repo.Create(t.Context(), globexOwner, note))

	err := repo.Delete(t.Context(), acmeOwner, note.ID)

	require.ErrorIs(t, err, domain.ErrNoteNotFound)

	got, err := repo.Get(t.Context(), globex, note.ID)
	require.NoError(t, err)
	require.Equal(t, "da Globex", got.Title)
}

func TestComponentRepository_RowLevelSecurityHidesOtherOrganizationsWithoutWhere(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)
	at := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	require.NoError(t, repo.Create(t.Context(), acmeOwner, newNote("da Acme", at)))
	require.NoError(t, repo.Create(t.Context(), globexOwner, newGlobexNote(at)))

	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)

	defer func() { _ = tx.Rollback(t.Context()) }()

	_, err = tx.Exec(t.Context(), `SELECT set_config('app.organization_id', $1, true)`, acme.UUID().String())
	require.NoError(t, err)

	rows, err := tx.Query(t.Context(), `SELECT title FROM notes`)
	require.NoError(t, err)

	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	require.Equal(t, []string{"da Acme"}, got)
}

func TestComponentRepository_TransactionWithoutOrganizationSeesNoNotes(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)
	at := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	require.NoError(t, repo.Create(t.Context(), acmeOwner, newNote("da Acme", at)))
	require.NoError(t, repo.Create(t.Context(), globexOwner, newGlobexNote(at)))

	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)

	defer func() { _ = tx.Rollback(t.Context()) }()

	var count int
	require.NoError(t, tx.QueryRow(t.Context(), `SELECT count(*) FROM notes`).Scan(&count))
	require.Zero(t, count)
}

func newRepository(t *testing.T) *notes.Repository {
	t.Helper()

	repo, _ := newRepositoryWithPool(t)

	return repo
}

func newRepositoryWithPool(t *testing.T, opts ...notes.Option) (*notes.Repository, *pgxpool.Pool) {
	t.Helper()

	if testing.Short() {
		t.Skip("component test: needs Postgres")
	}

	pool := testdb.New(t, migrations.FS)

	return notes.New(pool, schemaIDs, opts...), pool
}

func newNote(title string, createdAt time.Time) *domain.Note {
	note := fixture.RandomNote()
	note.Title = title
	note.CreatedAt = createdAt
	note.OrganizationID = acme

	return note
}

func newGlobexNote(createdAt time.Time) *domain.Note {
	note := newNote("da Globex", createdAt)
	note.OrganizationID = globex

	return note
}

func titles(list []*domain.Note) []string {
	out := make([]string, 0, len(list))
	for _, note := range list {
		out = append(out, note.Title)
	}

	return out
}
