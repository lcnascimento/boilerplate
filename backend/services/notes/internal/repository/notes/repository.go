package notes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes/internal/db"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

var ErrQuery = errors.New("notes query failed")

// Repository emits a NoteCreated or NoteDeleted Integration Event in the transaction of every effective mutation.
type Repository struct {
	pool      *pgxpool.Pool
	queries   *db.Queries
	schemaIDs events.SchemaIDs
	now       func() time.Time
}

func New(pool *pgxpool.Pool, schemaIDs events.SchemaIDs, opts ...Option) *Repository {
	r := &Repository{pool: pool, queries: db.New(pool), schemaIDs: schemaIDs, now: time.Now}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Repository) Create(ctx context.Context, note *domain.Note) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		err := r.queries.WithTx(tx).CreateNote(ctx, db.CreateNoteParams{
			ID:        note.ID.UUID(),
			CreatedAt: note.CreatedAt,
			Title:     note.Title,
			Body:      note.Body,
		})
		if err != nil {
			return ErrQuery.WithCause(err)
		}

		return r.emit(ctx, tx, note.ID, note.CreatedAt, &events.NoteCreated{
			Id:        note.ID.String(),
			Title:     note.Title,
			Body:      note.Body,
			CreatedAt: timestamppb.New(note.CreatedAt),
		})
	})
}

func (r *Repository) Get(ctx context.Context, id domain.NoteID) (*domain.Note, error) {
	row, err := r.queries.GetNote(ctx, id.UUID())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNoteNotFound
	}

	if err != nil {
		return nil, ErrQuery.WithCause(err)
	}

	return toDomain(row), nil
}

func (r *Repository) List(ctx context.Context) ([]*domain.Note, error) {
	rows, err := r.queries.ListNotes(ctx)
	if err != nil {
		return nil, ErrQuery.WithCause(err)
	}

	out := make([]*domain.Note, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}

	return out, nil
}

// Delete holds the row lock until commit, so of concurrent deletes of one Note only the first emits NoteDeleted.
func (r *Repository) Delete(ctx context.Context, id domain.NoteID) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		deleted, err := r.queries.WithTx(tx).DeleteNote(ctx, id.UUID())
		if err != nil {
			return ErrQuery.WithCause(err)
		}

		if deleted == 0 {
			return domain.ErrNoteNotFound
		}

		return r.emit(ctx, tx, id, r.now(), &events.NoteDeleted{Id: id.String()})
	})
}

func (r *Repository) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ErrQuery.WithCause(err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return ErrQuery.WithCause(err)
	}

	return nil
}

// toDomain returns CreatedAt in UTC: pgx reads timestamptz in the local zone.
func toDomain(row db.Note) *domain.Note {
	return &domain.Note{
		ID:          domain.NoteIDFromUUID(row.ID),
		NoteContent: domain.NoteContent{Title: row.Title, Body: row.Body},
		CreatedAt:   row.CreatedAt.UTC(),
	}
}
