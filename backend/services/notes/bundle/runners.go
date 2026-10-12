package bundle

import (
	"context"
	"log/slog"
	"os"

	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/o11y"
	"github.com/lcnascimento/go-kit/o11y/log"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend"
)

var logger = newLogger()

func newLogger() *slog.Logger {
	handler, err := log.NewHandler(pkg)
	if err != nil {
		panic(err)
	}

	return slog.New(handler)
}

func Run(app *fx.App) {
	if code := run(app); code != 0 {
		os.Exit(code)
	}
}

func run(app *fx.App) int {
	ctx := context.Background()

	o11y.MustStart(ctx)

	defer func() {
		if err := o11y.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "failed to shutdown o11y", "error", err)
		}
	}()

	if err := app.Start(ctx); err != nil {
		logStartFailure(ctx, err)

		return 1
	}

	defer func() {
		if err := app.Stop(ctx); err != nil {
			logger.ErrorContext(ctx, "failed to stop app", "error", err)
		}
	}()

	signal := <-app.Wait()

	return signal.ExitCode
}

// logStartFailure surfaces the error a constructor returned, with its cause,
// which fx's own report buries under the dependency chain.
func logStartFailure(ctx context.Context, err error) {
	var cause errors.CustomError
	if !errors.As(err, &cause) {
		return
	}

	logger.ErrorContext(ctx, cause.Error(), "reasons", errors.Reasons(cause), "attributes", errors.Attributes(cause))
}

func NewFrontendServer() *fx.App {
	return fx.New(frontendServerOptions()...)
}

func frontendServerOptions() []fx.Option {
	return []fx.Option{
		Logger(logger),
		Validator(),
		Database(),
		Repositories(),
		NotesService(),
		fx.Provide(func(notes transport.NotesService, shutdowner fx.Shutdowner) *frontend.Server {
			return frontend.New(notes, frontend.WithShutdown(func() {
				_ = shutdowner.Shutdown(fx.ExitCode(1))
			}))
		}),
		fx.Invoke(func(lifecycle fx.Lifecycle, server *frontend.Server) {
			lifecycle.Append(fx.Hook{
				OnStart: server.Start,
				OnStop:  server.Shutdown,
			})
		}),
	}
}
