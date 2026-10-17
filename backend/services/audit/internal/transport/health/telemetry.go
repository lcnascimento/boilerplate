package health

import (
	"context"

	"github.com/lcnascimento/go-kit/o11y/log"
)

var logger = log.MustNewLogger("audit/transport/health")

func onUnhealthy(ctx context.Context, err error) {
	logger.Error(ctx, err)
}
