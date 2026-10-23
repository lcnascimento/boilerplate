package bundle

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/o11y"
)

func Run(newAppAfterO11yStarts func() *fx.App) {
	if code := run(newAppAfterO11yStarts); code != 0 {
		os.Exit(code)
	}
}

func run(newAppAfterO11yStarts func() *fx.App) int {
	ctx := context.Background()

	o11y.MustStart(ctx)

	defer func() {
		if err := o11y.Shutdown(ctx); err != nil {
			onO11yShutdownFailed(ctx, err)
		}
	}()

	app := newAppAfterO11yStarts()

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
