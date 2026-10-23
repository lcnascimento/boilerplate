package consumer

import (
	"context"

	"github.com/lcnascimento/go-kit/o11y/log"
)

var logger = log.MustNewLogger("notes/transport/consumer")

func onInvalidEvent(ctx context.Context, err error) error {
	logger.ErrorBySeverity(ctx, err)

	return err
}
