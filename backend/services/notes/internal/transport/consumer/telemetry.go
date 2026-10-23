package consumer

import (
	"context"

	"github.com/lcnascimento/go-kit/o11y/log"
)

var logger = log.MustNewLogger("notes/transport/consumer")

func onRouterStopped(ctx context.Context, err error) {
	logger.Error(ctx, err)
}
