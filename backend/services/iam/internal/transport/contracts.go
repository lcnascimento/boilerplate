package transport

import (
	"context"
	"time"
)

type ChangesService interface {
	Translate(ctx context.Context) (int, error)
	OldestPendingAge(ctx context.Context) (time.Duration, error)
}

type Database interface {
	Ping(ctx context.Context) error
}
