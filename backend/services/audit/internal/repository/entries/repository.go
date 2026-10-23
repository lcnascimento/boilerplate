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
	err := r.queries.InsertAuditEntry(ctx, db.InsertAuditEntryParams{
		Source:          entry.Source,
		ID:              entry.ID,
		Type:            entry.Type,
		SpecVersion:     entry.SpecVersion,
		Subject:         nullable(entry.Subject),
		Time:            entry.Time,
		DataContentType: entry.DataContentType,
		Headers:         entry.Headers,
		Payload:         entry.Payload,
		KafkaTopic:      entry.KafkaTopic,
		KafkaPartition:  entry.KafkaPartition,
		KafkaOffset:     entry.KafkaOffset,

		ActorID:             nullable(entry.ActorID),
		ActorOrganizationID: nullable(entry.ActorOrganizationID),
	})
	if err != nil {
		return ErrQuery.WithCause(err)
	}

	return nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
