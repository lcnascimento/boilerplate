package forwarder

import (
	"context"
	"time"

	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
)

const purgeInterval = time.Hour

// Janitor purges the outbox on start and then every purgeInterval, and reports the age of its oldest
// pending message. Every replica runs one.
type Janitor struct {
	outbox   transport.OutboxService
	interval time.Duration

	cancel       context.CancelFunc
	done         chan struct{}
	registration otelmetric.Registration
}

type JanitorOption func(*Janitor)

func WithPurgeInterval(interval time.Duration) JanitorOption {
	return func(j *Janitor) {
		j.interval = interval
	}
}

func NewJanitor(outbox transport.OutboxService, opts ...JanitorOption) *Janitor {
	j := &Janitor{outbox: outbox, interval: purgeInterval}

	for _, opt := range opts {
		opt(j)
	}

	return j
}

func (j *Janitor) Start(ctx context.Context) error {
	registration, err := registerOldestPendingAge(j.outbox.OldestPendingAge)
	if err != nil {
		return err
	}

	j.registration = registration

	ctx, j.cancel = context.WithCancel(context.WithoutCancel(ctx))
	j.done = make(chan struct{})

	go j.run(ctx)

	return nil
}

func (j *Janitor) Stop(ctx context.Context) error {
	j.cancel()

	select {
	case <-j.done:
	case <-ctx.Done():
		return ctx.Err()
	}

	return j.registration.Unregister()
}

// run leaves failures to the next tick: Purge already reports them.
func (j *Janitor) run(ctx context.Context) {
	defer close(j.done)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		_ = j.outbox.Purge(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
