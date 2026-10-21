package bundle

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/o11y"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/consumer"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/health"
)

// Run starts o11y before newApp builds the graph, so what constructors record reaches the exporters.
func Run(newApp func() *fx.App) {
	if code := run(newApp); code != 0 {
		os.Exit(code)
	}
}

func run(newApp func() *fx.App) int {
	ctx := context.Background()

	o11y.MustStart(ctx)

	defer func() {
		if err := o11y.Shutdown(ctx); err != nil {
			onO11yShutdownFailed(ctx, err)
		}
	}()

	app := newApp()

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

func NewConsumer() *fx.App {
	return fx.New(consumerOptions()...)
}

func consumerOptions() []fx.Option {
	return []fx.Option{
		Logger(logger),
		Validator(),
		Database(),
		Repositories(),
		AuditService(),
		fx.Provide(newConsumer, newHealthServer),
		fx.Invoke(func(lifecycle fx.Lifecycle, consumer *consumer.Consumer, server *health.Server) {
			lifecycle.Append(fx.Hook{OnStart: consumer.Start, OnStop: consumer.Stop})
			lifecycle.Append(fx.Hook{OnStart: server.Start, OnStop: server.Shutdown})
		}),
	}
}

func newConsumer(audit transport.AuditService, shutdowner fx.Shutdowner) (*consumer.Consumer, error) {
	brokers, err := config.KafkaBrokers()
	if err != nil {
		return nil, err
	}

	return consumer.New(audit, brokers, consumer.WithShutdown(exitWithFailure(shutdowner))), nil
}

func newHealthServer(db transport.Database, c *consumer.Consumer, shutdowner fx.Shutdowner) (*health.Server, error) {
	port, err := config.HealthPort()
	if err != nil {
		return nil, err
	}

	return health.New(db, c, port, exitWithFailure(shutdowner)), nil
}

func exitWithFailure(shutdowner fx.Shutdowner) func() {
	return func() { _ = shutdowner.Shutdown(fx.ExitCode(1)) }
}
