package bundle

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/o11y"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend"
)

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
			onO11yShutdownFailed(ctx, err)
		}
	}()

	// fx only listens for signals from app.Wait on.
	startCtx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	err := app.Start(startCtx)

	stopSignals()

	if err != nil {
		onStartFailed(ctx, err)

		return 1
	}

	defer func() {
		if err := app.Stop(ctx); err != nil {
			onStopFailed(ctx, err)
		}
	}()

	shutdown := <-app.Wait()

	return shutdown.ExitCode
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
		ActorMiddleware(),
		fx.Provide(func(
			notes transport.NotesService,
			db transport.Database,
			authenticate mux.MiddlewareFunc,
			shutdowner fx.Shutdowner,
		) *frontend.Server {
			return frontend.New(notes, db, authenticate, frontend.WithShutdown(func() {
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
