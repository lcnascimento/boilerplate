package eventbus

import (
	"context"
	"sync/atomic"
	"time"
)

const consumerRunningTimeout = 30 * time.Second

type Consumer struct {
	config   SubscriberConfig
	shutdown func()

	subscriber atomic.Pointer[Subscriber]
	stopped    atomic.Bool
	closing    atomic.Bool
	done       chan struct{}
}

type ConsumerOption func(*Consumer)

func WithShutdown(shutdown func()) ConsumerOption {
	return func(c *Consumer) {
		c.shutdown = shutdown
	}
}

func NewConsumer(config *SubscriberConfig, opts ...ConsumerOption) *Consumer {
	c := &Consumer{config: *config, shutdown: func() {}, done: make(chan struct{})}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Consumer) Start(ctx context.Context) error {
	subscriber, err := NewSubscriber(&c.config)
	if err != nil {
		return err
	}

	c.subscriber.Store(subscriber)

	go c.run(context.WithoutCancel(ctx), subscriber)

	ctx, cancel := context.WithTimeout(ctx, consumerRunningTimeout)
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

func (c *Consumer) run(ctx context.Context, subscriber *Subscriber) {
	defer close(c.done)

	err := subscriber.Run(ctx)

	c.stopped.Store(true)

	if !c.closing.Load() {
		onRouterStopped(ctx, ErrRouterStopped.WithCause(err))
		c.shutdown()
	}
}
