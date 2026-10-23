package notes_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

func TestComponentRepository_CreateInADeletedOrganizationIsRefused(t *testing.T) {
	t.Parallel()

	repo := newRepository(t)
	at := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)

	_, err := repo.DeleteByOrganization(t.Context(), acme)
	require.NoError(t, err)

	err = repo.Create(t.Context(), acmeOwner, newNote("depois da exclusão", at))
	require.ErrorIs(t, err, domain.ErrOrganizationDeleted)

	require.NoError(t, repo.Create(t.Context(), globexOwner, newGlobexNote(at)))

	acmeNotes, err := repo.List(t.Context(), acme)
	require.NoError(t, err)
	require.Empty(t, acmeNotes)
}

func TestComponentRepository_CreateDuringTheDeletionOfItsOrganizationLeavesNoNote(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)
	paused, release := make(chan struct{}), make(chan struct{})
	deleting := notes.New(pausingPool(t, pool, "commit", paused, release), schemaIDs)

	var wg sync.WaitGroup

	wg.Go(func() {
		_, err := deleting.DeleteByOrganization(t.Context(), acme)
		assert.NoError(t, err)
	})

	<-paused

	created := make(chan error, 1)

	go func() {
		created <- repo.Create(t.Context(), acmeOwner, newNote("durante a exclusão", time.Now()))
	}()

	select {
	case err := <-created:
		close(release)
		wg.Wait()
		require.ErrorIs(t, err, domain.ErrOrganizationDeleted)
	case <-time.After(time.Second):
		close(release)
		wg.Wait()
		require.ErrorIs(t, <-created, domain.ErrOrganizationDeleted)
	}

	acmeNotes, err := repo.List(t.Context(), acme)
	require.NoError(t, err)
	require.Empty(t, acmeNotes)
}

type pauseOn struct {
	sql     string
	paused  chan<- struct{}
	release <-chan struct{}
}

func (p pauseOn) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if strings.EqualFold(data.SQL, p.sql) {
		close(p.paused)
		<-p.release
	}

	return ctx
}

func (pauseOn) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func pausingPool(t *testing.T, pool *pgxpool.Pool, sql string, paused chan<- struct{}, release <-chan struct{}) *pgxpool.Pool {
	t.Helper()

	cfg := pool.Config()
	cfg.ConnConfig.Tracer = pauseOn{sql: sql, paused: paused, release: release}

	pausing, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(pausing.Close)

	return pausing
}
