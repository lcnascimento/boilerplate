package bundle

import (
	"context"
	"maps"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/sr"
	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/o11y"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const (
	schemasTimeout      = time.Minute
	schemaLookupTimeout = 10 * time.Second
)

var ErrConnectSchemaRegistry = errors.New("failed to create the Schema Registry client: check SCHEMA_REGISTRY_URL").
	WithCode("CONNECT_SCHEMA_REGISTRY")

func RegisterSchemas() {
	if code := registerSchemas(); code != 0 {
		os.Exit(code)
	}
}

func registerSchemas() int {
	ctx := context.Background()

	o11y.MustStart(ctx)

	defer func() {
		if err := o11y.Shutdown(ctx); err != nil {
			onO11yShutdownFailed(ctx, err)
		}
	}()

	signalCtx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	registerCtx, cancel := context.WithTimeout(signalCtx, schemasTimeout)
	defer cancel()

	ids, err := register(registerCtx)
	if err != nil {
		onCommandFailed(ctx, err)

		return 1
	}

	onSchemasRegistered(ctx, ids)

	return 0
}

func register(ctx context.Context) (events.SchemaIDs, error) {
	client, err := newRegistryClient()
	if err != nil {
		return nil, err
	}

	return events.RegisterSchemas(ctx, client)
}

func newSchemaIDs(lifecycle fx.Lifecycle) (events.SchemaIDs, error) {
	client, err := newRegistryClient()
	if err != nil {
		return nil, err
	}

	ids := events.SchemaIDs{}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, schemaLookupTimeout)
			defer cancel()

			found, err := events.LookupSchemaIDs(ctx, client)
			if err != nil {
				return err
			}

			maps.Copy(ids, found)

			return nil
		},
	})

	return ids, nil
}

func newRegistryClient() (*sr.Client, error) {
	url, err := config.SchemaRegistryURL()
	if err != nil {
		return nil, err
	}

	client, err := sr.NewClient(sr.URLs(url))
	if err != nil {
		return nil, ErrConnectSchemaRegistry.WithCause(err)
	}

	return client, nil
}
