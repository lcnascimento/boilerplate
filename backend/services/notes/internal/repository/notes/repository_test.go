package notes_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
)

func TestComponentRepository_StartsEmpty(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)

	got, err := repo.List(t.Context())

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestComponentRepository_CreateThenGet(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 123456000, time.UTC))

	require.NoError(t, repo.Create(t.Context(), note))

	got, err := repo.Get(t.Context(), note.ID)

	require.NoError(t, err)
	require.Equal(t, note, got)
}

func TestComponentRepository_GetUnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)

	_, err := repo.Get(t.Context(), domain.NewNoteID())

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
		require.NoError(t, repo.Create(t.Context(), note))
	}

	got, err := repo.List(t.Context())

	require.NoError(t, err)
	require.Equal(t, []string{"newest", "middle", "oldest"}, titles(got))
}

func TestComponentRepository_ListBreaksTimestampTiesByID(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	at := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	first := newNote("first", at)
	second := newNote("second", at)

	require.NoError(t, repo.Create(t.Context(), first))
	require.NoError(t, repo.Create(t.Context(), second))

	got, err := repo.List(t.Context())

	require.NoError(t, err)
	require.Equal(t, []string{"second", "first"}, titles(got))
}

func TestComponentRepository_DeleteRemovesNote(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	require.NoError(t, repo.Create(t.Context(), note))

	require.NoError(t, repo.Delete(t.Context(), note.ID))

	_, err := repo.Get(t.Context(), note.ID)
	require.ErrorIs(t, err, domain.ErrNoteNotFound)
}

func TestComponentRepository_DeleteUnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)

	err := repo.Delete(t.Context(), domain.NewNoteID())

	require.ErrorIs(t, err, domain.ErrNoteNotFound)
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

	return note
}

func titles(list []*domain.Note) []string {
	out := make([]string, 0, len(list))
	for _, note := range list {
		out = append(out, note.Title)
	}

	return out
}
