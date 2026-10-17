package notes

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes/internal/db"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

var ErrQuery = errors.New("notes query failed")

type Repository struct {
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{queries: db.New(pool)}
}

func (r *Repository) Create(ctx context.Context, note *domain.Note) error {
	err := r.queries.CreateNote(ctx, db.CreateNoteParams{
		ID:        note.ID.UUID(),
		CreatedAt: note.CreatedAt,
		Title:     note.Title,
		Body:      note.Body,
	})
	if err != nil {
		return ErrQuery.WithCause(err)
	}

	return nil
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

func (r *Repository) Delete(ctx context.Context, id domain.NoteID) error {
	deleted, err := r.queries.DeleteNote(ctx, id.UUID())
	if err != nil {
		return ErrQuery.WithCause(err)
	}

	if deleted == 0 {
		return domain.ErrNoteNotFound
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
