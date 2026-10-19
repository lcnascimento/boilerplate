package consumer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport"
)

const (
	Group = "audit"

	runningTimeout = 30 * time.Second
)

var (
	ErrNotRunning = errors.New("events subscriber did not start consuming in time").
			WithCode("SUBSCRIBER_NOT_RUNNING")

	ErrRouterStopped = errors.New("events subscriber stopped while the process was running").
				WithCode("ROUTER_STOPPED")
)

type Consumer struct {
	events.UnimplementedObserver

	audit    transport.AuditService
	brokers  []string
	shutdown func()

	subscriber atomic.Pointer[events.Subscriber]
	stopped    atomic.Bool
	closing    atomic.Bool
	done       chan struct{}
}

func New(audit transport.AuditService, brokers []string, opts ...Option) *Consumer {
	c := &Consumer{audit: audit, brokers: brokers, shutdown: func() {}, done: make(chan struct{})}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Consumer) OnAny(ctx context.Context, event *events.Event) error {
	return c.audit.Record(ctx, &idomain.Entry{
		Source:          event.Source,
		ID:              event.ID,
		Type:            event.Type,
		SpecVersion:     event.SpecVersion,
		Subject:         event.Subject,
		Time:            event.Time,
		DataContentType: event.DataContentType,
		Headers:         event.Headers,
		Payload:         event.Payload,
		KafkaTopic:      event.Topic,
		KafkaPartition:  event.Partition,
		KafkaOffset:     event.Offset,
	})
}

func (c *Consumer) Start(ctx context.Context) error {
	subscriber, err := events.NewSubscriber(events.SubscriberConfig{Brokers: c.brokers, Group: Group, Observer: c})
	if err != nil {
		return err
	}

	c.subscriber.Store(subscriber)

	go c.run(context.WithoutCancel(ctx), subscriber)

	ctx, cancel := context.WithTimeout(ctx, runningTimeout)
	defer cancel()

	select {
	case <-subscriber.Running():
		return nil
	case <-c.done:
		return ErrRouterStopped
	case <-ctx.Done():
		_ = c.Stop(ctx)

		return ErrNotRunning.WithCause(ctx.Err())
	}
}

func (c *Consumer) Stop(_ context.Context) error {
	c.closing.Store(true)

	subscriber := c.subscriber.Load()
	if subscriber == nil {
		return nil
	}

	return subscriber.Close()
}

func (c *Consumer) Running() bool {
	subscriber := c.subscriber.Load()
	if subscriber == nil || c.stopped.Load() {
		return false
	}

	select {
	case <-subscriber.Running():
		return true
	default:
		return false
	}
}

func (c *Consumer) run(ctx context.Context, subscriber *events.Subscriber) {
	defer close(c.done)

	err := subscriber.Run(ctx)

	c.stopped.Store(true)

	if !c.closing.Load() {
		onRouterStopped(ctx, ErrRouterStopped.WithCause(err))
		c.shutdown()
	}
}
