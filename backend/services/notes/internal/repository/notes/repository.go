package notes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes/internal/db"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

var ErrQuery = errors.New("notes query failed")

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
	return r.inTx(ctx, note.OrganizationID, func(tx pgx.Tx, queries *db.Queries) error {
		err := queries.CreateNote(ctx, db.CreateNoteParams{
			ID:             note.ID.UUID(),
			OrganizationID: note.OrganizationID.UUID(),
			CreatedAt:      note.CreatedAt,
			Title:          note.Title,
			Body:           note.Body,
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

func (r *Repository) Get(ctx context.Context, orgID iamdomain.OrganizationID, id domain.NoteID) (*domain.Note, error) {
	var note *domain.Note

	err := r.inTx(ctx, orgID, func(_ pgx.Tx, queries *db.Queries) error {
		row, err := queries.GetNote(ctx, db.GetNoteParams{OrganizationID: orgID.UUID(), ID: id.UUID()})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNoteNotFound
		}

		if err != nil {
			return ErrQuery.WithCause(err)
		}

		note = toDomain(&row)

		return nil
	})

	return note, err
}

func (r *Repository) List(ctx context.Context, orgID iamdomain.OrganizationID) ([]*domain.Note, error) {
	var out []*domain.Note

	err := r.inTx(ctx, orgID, func(_ pgx.Tx, queries *db.Queries) error {
		rows, err := queries.ListNotes(ctx, orgID.UUID())
		if err != nil {
			return ErrQuery.WithCause(err)
		}

		out = make([]*domain.Note, 0, len(rows))
		for _, row := range rows {
			out = append(out, toDomain(&row))
		}

		return nil
	})

	return out, err
}

func (r *Repository) Delete(ctx context.Context, orgID iamdomain.OrganizationID, id domain.NoteID) error {
	return r.inTx(ctx, orgID, func(tx pgx.Tx, queries *db.Queries) error {
		deleted, err := queries.DeleteNote(ctx, db.DeleteNoteParams{OrganizationID: orgID.UUID(), ID: id.UUID()})
		if err != nil {
			return ErrQuery.WithCause(err)
		}

		if deleted == 0 {
			return domain.ErrNoteNotFound
		}

		return r.emit(ctx, tx, id, r.now(), &events.NoteDeleted{Id: id.String()})
	})
}

// inTx scopes the transaction to orgID, so the notes row-level security policy only shows that Organization's rows.
func (r *Repository) inTx(ctx context.Context, orgID iamdomain.OrganizationID, fn func(pgx.Tx, *db.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ErrQuery.WithCause(err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	if err := queries.SetOrganization(ctx, orgID.UUID().String()); err != nil {
		return ErrQuery.WithCause(err)
	}

	if err := fn(tx, queries); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return ErrQuery.WithCause(err)
	}

	return nil
}

// toDomain returns CreatedAt in UTC: pgx reads timestamptz in the local zone.
func toDomain(row *db.Note) *domain.Note {
	return &domain.Note{
		ID:             domain.NoteIDFromUUID(row.ID),
		NoteContent:    domain.NoteContent{Title: row.Title, Body: row.Body},
		OrganizationID: iamdomain.OrganizationIDFromUUID(row.OrganizationID),
		CreatedAt:      row.CreatedAt.UTC(),
	}
}
