package bundle

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/sr"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/o11y"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const schemasTimeout = time.Minute

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
	url, err := config.SchemaRegistryURL()
	if err != nil {
		return nil, err
	}

	client, err := sr.NewClient(sr.URLs(url))
	if err != nil {
		return nil, ErrConnectSchemaRegistry.WithCause(err)
	}

	return events.RegisterSchemas(ctx, client)
}
