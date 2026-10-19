package entries

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/repository/entries/internal/db"
)

var ErrQuery = errors.New("audit entries query failed")

type Repository struct {
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{queries: db.New(pool)}
}

func (r *Repository) Insert(ctx context.Context, entry *idomain.Entry) error {
	var subject *string
	if entry.Subject != "" {
		subject = &entry.Subject
	}

	err := r.queries.InsertAuditEntry(ctx, db.InsertAuditEntryParams{
		Source:          entry.Source,
		ID:              entry.ID,
		Type:            entry.Type,
		SpecVersion:     entry.SpecVersion,
		Subject:         subject,
		Time:            entry.Time,
		DataContentType: entry.DataContentType,
		Headers:         entry.Headers,
		Payload:         entry.Payload,
		KafkaTopic:      entry.KafkaTopic,
		KafkaPartition:  entry.KafkaPartition,
		KafkaOffset:     entry.KafkaOffset,
	})
	if err != nil {
		return ErrQuery.WithCause(err)
	}

	return nil
}
