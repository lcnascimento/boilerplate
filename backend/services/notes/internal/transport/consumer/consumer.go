package consumer

import (
	"context"
	"sync/atomic"
	"time"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	iamevents "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
)

const (
	Group = "notes"

	runningTimeout = 30 * time.Second
)

var Topics = []string{iamevents.OrganizationTopic}

var (
	ErrNotRunning = errors.New("events subscriber did not start consuming in time").
			WithCode("SUBSCRIBER_NOT_RUNNING")

	ErrRouterStopped = errors.New("events subscriber stopped while the process was running").
				WithCode("ROUTER_STOPPED")

	ErrNotConsuming = errors.New("events subscriber is not consuming").WithCode("SUBSCRIBER_NOT_CONSUMING")
)

type Consumer struct {
	notes    transport.OrganizationNotesService
	brokers  []string
	shutdown func()

	subscriber atomic.Pointer[eventbus.Subscriber]
	stopped    atomic.Bool
	closing    atomic.Bool
	done       chan struct{}
}

func New(notes transport.OrganizationNotesService, brokers []string, opts ...Option) *Consumer {
	c := &Consumer{notes: notes, brokers: brokers, shutdown: func() {}, done: make(chan struct{})}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Consumer) Handle(ctx context.Context, event *eventbus.Event) error {
	deleted, ok := event.Message.(*iamevents.OrganizationDeleted)
	if !ok {
		return nil
	}

	orgID, err := iamdomain.ParseOrganizationID(deleted.GetOrganizationId())
	if err != nil {
		return eventbus.ErrPermanent.WithCause(err)
	}

	return c.notes.DeleteOrganizationNotes(ctx, orgID)
}

func (c *Consumer) Start(ctx context.Context) error {
	subscriber, err := eventbus.NewSubscriber(&eventbus.SubscriberConfig{Brokers: c.brokers, Group: Group, Topics: Topics, Handler: c})
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

func (c *Consumer) Ready(ctx context.Context) error {
	subscriber := c.subscriber.Load()
	if subscriber == nil || c.stopped.Load() {
		return ErrNotConsuming
	}

	select {
	case <-subscriber.Running():
		return subscriber.Ping(ctx)
	default:
		return ErrNotConsuming
	}
}

func (c *Consumer) run(ctx context.Context, subscriber *eventbus.Subscriber) {
	defer close(c.done)

	err := subscriber.Run(ctx)

	c.stopped.Store(true)

	if !c.closing.Load() {
		onRouterStopped(ctx, ErrRouterStopped.WithCause(err))
		c.shutdown()
	}
}
