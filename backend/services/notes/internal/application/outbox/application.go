package outbox

import (
	"context"
	"time"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application"
)

const retention = 7 * 24 * time.Hour

type Application struct {
	outbox application.OutboxRepository
}

func New(outbox application.OutboxRepository) *Application {
	return &Application{outbox: outbox}
}

func (a *Application) Purge(ctx context.Context) error {
	ctx, span := a.onPurgeStart(ctx)
	defer span.End()

	purged, err := a.outbox.Purge(ctx, retention)
	if err != nil {
		return a.onPurgeFailed(ctx, purged, ErrPurge.WithCause(err))
	}

	a.onPurged(ctx, purged)

	return nil
}

func (a *Application) OldestPendingAge(ctx context.Context) (time.Duration, error) {
	age, err := a.outbox.OldestPendingAge(ctx)
	if err != nil {
		return 0, ErrOldestPendingAge.WithCause(err)
	}

	return age, nil
}
