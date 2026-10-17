package outbox

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/outbox/internal/db"
)

const purgeBatchSize = 10_000

var ErrQuery = errors.New("outbox query failed")

type Repository struct {
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{queries: db.New(pool)}
}

// Purge deletes, in batches, the messages ConsumerGroup has already forwarded that are older than retention.
// It returns how many it deleted, including those of the batches before a failure.
func (r *Repository) Purge(ctx context.Context, retention time.Duration) (int64, error) {
	var total int64

	for {
		deleted, err := r.queries.PurgeForwarded(ctx, db.PurgeForwardedParams{
			ConsumerGroup:    ConsumerGroup,
			RetentionSeconds: retention.Seconds(),
			BatchSize:        purgeBatchSize,
		})
		if err != nil {
			return total, ErrQuery.WithCause(err)
		}

		total += deleted

		if deleted < purgeBatchSize {
			return total, nil
		}
	}
}

// OldestPendingAge is zero when ConsumerGroup has forwarded every message.
func (r *Repository) OldestPendingAge(ctx context.Context) (time.Duration, error) {
	seconds, err := r.queries.OldestPendingAgeSeconds(ctx, ConsumerGroup)
	if err != nil {
		return 0, ErrQuery.WithCause(err)
	}

	return time.Duration(seconds * float64(time.Second)), nil
}
