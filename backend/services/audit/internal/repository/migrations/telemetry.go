package migrations

import (
	"context"

	"github.com/pressly/goose/v3"

	"github.com/lcnascimento/go-kit/o11y/log"
)

var logger = log.MustNewLogger("audit/repository/migrations")

func onMigrated(ctx context.Context, result *goose.MigrationResult) {
	attrs := []log.Attr{
		log.String("direction", result.Direction),
		log.Int("version", int(result.Source.Version)),
		log.String("file", result.Source.Path),
		log.String("duration", result.Duration.String()),
	}

	if result.Error != nil {
		logger.Error(ctx, ErrMigrate.WithCause(result.Error), attrs...)

		return
	}

	logger.Info(ctx, "migration "+result.Direction, attrs...)
}
