package health

import (
	"context"

	"github.com/lcnascimento/go-kit/o11y/log"
)

var logger = log.MustNewLogger("notes/transport/health")

// onUnhealthy logs the ping failure; the response body never carries it.
func onUnhealthy(ctx context.Context, err error) {
	logger.Error(ctx, ErrDatabaseUnhealthy.WithCause(err))
}
