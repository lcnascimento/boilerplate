package application

import (
	"context"
	"time"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/idomain"
)

type ChangesRepository interface {
	Translate(ctx context.Context, limit int32) ([]*idomain.Event, error)
	OldestPendingAge(ctx context.Context) (time.Duration, error)
}
