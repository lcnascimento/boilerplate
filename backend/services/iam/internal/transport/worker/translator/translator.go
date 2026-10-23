package translator

import (
	"context"
	"time"

	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport"
)

const defaultPollInterval = time.Second

type Translator struct {
	changes      transport.ChangesService
	pollInterval time.Duration

	cancel       context.CancelFunc
	done         chan struct{}
	registration otelmetric.Registration
}

func New(changes transport.ChangesService, opts ...Option) *Translator {
	t := &Translator{changes: changes, pollInterval: defaultPollInterval}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (t *Translator) Start(ctx context.Context) error {
	registration, err := t.registerOldestPendingAge()
	if err != nil {
		return ErrStart.WithCause(err)
	}

	t.registration = registration

	ctx, t.cancel = context.WithCancel(context.WithoutCancel(ctx))
	t.done = make(chan struct{})

	go t.run(ctx)

	return nil
}

func (t *Translator) Stop(ctx context.Context) error {
	t.cancel()

	unregisterErr := t.registration.Unregister()

	select {
	case <-t.done:
	case <-ctx.Done():
		return ctx.Err()
	}

	if unregisterErr != nil {
		return ErrStop.WithCause(unregisterErr)
	}

	return nil
}

func (t *Translator) run(ctx context.Context) {
	defer close(t.done)

	poll := time.NewTimer(0)
	defer poll.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-poll.C:
		}

		t.drain(ctx)
		poll.Reset(t.pollInterval)
	}
}

func (t *Translator) drain(ctx context.Context) {
	for {
		translated, err := t.changes.Translate(ctx)
		if err != nil || translated == 0 {
			return
		}
	}
}
