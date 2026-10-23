package bundle

import (
	"context"
	"log/slog"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/o11y/log"
)

var logger = newLogger()

func newLogger() *slog.Logger {
	handler, err := log.NewHandler(pkg)
	if err != nil {
		panic(err)
	}

	return slog.New(handler)
}

// fx's own report buries the constructor's error under the dependency chain.
func onStartFailed(ctx context.Context, err error) {
	var cause errors.CustomError
	if !errors.As(err, &cause) {
		return
	}

	logger.ErrorContext(ctx, cause.Error(), "reasons", errors.Reasons(cause), "attributes", errors.Attributes(cause))
}

func onCommandFailed(ctx context.Context, err error) {
	logger.ErrorContext(ctx, err.Error(), "reasons", errors.Reasons(err), "attributes", errors.Attributes(err))
}

func onSchemasRegistered(ctx context.Context, ids map[string]int) {
	logger.InfoContext(ctx, "schemas registered", "ids", ids)
}

func onStopFailed(ctx context.Context, err error) {
	logger.ErrorContext(ctx, "failed to stop app", "error", err)
}

func onO11yShutdownFailed(ctx context.Context, err error) {
	logger.ErrorContext(ctx, "failed to shutdown o11y", "error", err)
}
