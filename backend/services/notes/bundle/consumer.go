package bundle

import (
	"go.uber.org/fx"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/consumer"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/healthserver"
)

func NewConsumer() *fx.App {
	return fx.New(consumerOptions()...)
}

func consumerOptions() []fx.Option {
	return []fx.Option{
		Logger(logger),
		Validator(),
		Database(),
		Repositories(),
		NotesService(),
		fx.Provide(newConsumer, newConsumerHealthServer),
		fx.Invoke(func(lifecycle fx.Lifecycle, c *consumer.Consumer, health *healthserver.Server) {
			lifecycle.Append(fx.Hook{OnStart: c.Start, OnStop: c.Stop})
			lifecycle.Append(fx.Hook{OnStart: health.Start, OnStop: health.Shutdown})
		}),
	}
}

func newConsumer(notes transport.OrganizationNotesService, shutdowner fx.Shutdowner) (*consumer.Consumer, error) {
	brokers, err := config.KafkaBrokers()
	if err != nil {
		return nil, err
	}

	return consumer.New(notes, brokers, consumer.WithShutdown(exitWithFailure(shutdowner))), nil
}

func newConsumerHealthServer(
	db transport.Database, c *consumer.Consumer, shutdowner fx.Shutdowner,
) (*healthserver.Server, error) {
	port, err := config.HealthPort()
	if err != nil {
		return nil, err
	}

	return healthserver.New(port, exitWithFailure(shutdowner), c.Ready, db.Ping), nil
}
