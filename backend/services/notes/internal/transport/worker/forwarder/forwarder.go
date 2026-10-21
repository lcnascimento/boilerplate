package forwarder

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/outbox"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const (
	startTimeout = 30 * time.Second
	closeTimeout = 30 * time.Second

	retryMaxRetries      = 10
	retryInitialInterval = 100 * time.Millisecond
	retryMaxInterval     = 30 * time.Second
	retryMultiplier      = 2
)

var (
	ErrKafkaUnavailable = errors.New("failed to connect to Kafka: is the broker up and KAFKA_BROKERS right?").
				WithCode("KAFKA_UNAVAILABLE")

	ErrTopicMissing = errors.New("topic does not exist: declare it with make kafka.topics").
			WithCode("TOPIC_MISSING")

	ErrInvalidConfig = errors.New("forwarder needs the outbox subscriber and Kafka brokers").
				WithCode("INVALID_FORWARDER_CONFIG")

	ErrStart = errors.New("forwarder did not start").WithCode("FORWARDER_START")

	ErrNotRunning = errors.New("forwarder is not running").WithCode("FORWARDER_NOT_RUNNING")

	ErrStop = errors.New("forwarder did not stop cleanly").WithCode("FORWARDER_STOP")
)

type Forwarder struct {
	router    *message.Router
	publisher *kafka.Publisher
	cluster   *events.Cluster
	shutdown  func()

	started  atomic.Bool
	stopping atomic.Bool
	stopOnce sync.Once
	stopErr  error
	runDone  chan struct{}
}

func New(outboxSubscriber message.Subscriber, brokers []string, opts ...Option) (*Forwarder, error) {
	if outboxSubscriber == nil || len(brokers) == 0 {
		return nil, ErrInvalidConfig
	}

	if err := checkTopic(brokers, events.Topic); err != nil {
		return nil, err
	}

	cluster, err := events.DialCluster(brokers, events.Topic)
	if err != nil {
		return nil, ErrKafkaUnavailable.WithCause(err)
	}

	publisher, err := newPublisher(brokers)
	if err != nil {
		_ = cluster.Close()

		return nil, err
	}

	f := &Forwarder{publisher: publisher, cluster: cluster, shutdown: func() {}, runDone: make(chan struct{})}
	for _, opt := range opts {
		opt(f)
	}

	f.router, err = message.NewRouter(message.RouterConfig{CloseTimeout: closeTimeout}, watermill.NopLogger{})
	if err != nil {
		f.close()

		return nil, ErrInvalidConfig.WithCause(err)
	}

	retry := middleware.Retry{
		MaxRetries:      retryMaxRetries,
		InitialInterval: retryInitialInterval,
		MaxInterval:     retryMaxInterval,
		Multiplier:      retryMultiplier,
	}

	_, err = forwarder.NewForwarder(outboxSubscriber, newTracedPublisher(publisher), watermill.NopLogger{}, forwarder.Config{
		ForwarderTopic: outbox.ForwarderTopic,
		Router:         f.router,
		Middlewares:    []message.HandlerMiddleware{retry.Middleware, middleware.Recoverer},
	})
	if err != nil {
		f.close()

		return nil, ErrInvalidConfig.WithCause(err)
	}

	return f, nil
}

func (f *Forwarder) Start(ctx context.Context) error {
	runErr := make(chan error, 1)

	f.started.Store(true)

	runCtx := context.WithoutCancel(ctx)

	go func() {
		defer close(f.runDone)

		err := f.router.Run(runCtx)
		runErr <- err

		if !f.stopping.Load() {
			onStoppedUnexpectedly(runCtx, err)
			f.shutdown()
		}
	}()

	timeout := time.NewTimer(startTimeout)
	defer timeout.Stop()

	select {
	case <-f.router.Running():
		onStarted(ctx)

		return nil
	case err := <-runErr:
		return ErrStart.WithCause(err)
	case <-timeout.C:
		return ErrStart.WithCause(context.DeadlineExceeded)
	case <-ctx.Done():
		return ErrStart.WithCause(ctx.Err())
	}
}

func (f *Forwarder) Stop(context.Context) error {
	f.stopOnce.Do(func() {
		f.stopping.Store(true)

		routerErr := f.router.Close()
		if f.started.Load() {
			<-f.runDone
		}

		publisherErr, clusterErr := f.publisher.Close(), f.cluster.Close()

		for _, err := range []error{routerErr, publisherErr, clusterErr} {
			if err != nil {
				f.stopErr = ErrStop.WithCause(err)

				return
			}
		}
	})

	return f.stopErr
}

func (f *Forwarder) Ready(ctx context.Context) error {
	if !f.router.IsRunning() || f.router.IsClosed() {
		return ErrNotRunning
	}

	if err := f.cluster.Ping(ctx); err != nil {
		return ErrKafkaUnavailable.WithCause(err)
	}

	return nil
}

func (f *Forwarder) close() {
	_ = f.publisher.Close()
	_ = f.cluster.Close()
}

func newPublisher(brokers []string) (*kafka.Publisher, error) {
	cfg := kafka.DefaultSaramaSyncPublisherConfig()
	cfg.Producer.Idempotent = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Net.MaxOpenRequests = 1

	publisher, err := kafka.NewPublisher(kafka.PublisherConfig{
		Brokers:               brokers,
		Marshaler:             kafka.NewWithPartitioningMarshaler(partitionKey),
		OverwriteSaramaConfig: cfg,
	}, watermill.NopLogger{})
	if err != nil {
		return nil, ErrKafkaUnavailable.WithCause(err)
	}

	return publisher, nil
}

func partitionKey(_ string, msg *message.Message) (string, error) {
	envelope, err := events.ParseEnvelope(msg.Metadata)
	if err != nil {
		return "", err
	}

	return envelope.PartitionKey, nil
}

func checkTopic(brokers []string, topic string) error {
	admin, err := sarama.NewClusterAdmin(brokers, sarama.NewConfig())
	if err != nil {
		return ErrKafkaUnavailable.WithCause(err)
	}

	defer func() { _ = admin.Close() }()

	topics, err := admin.ListTopics()
	if err != nil {
		return ErrKafkaUnavailable.WithCause(err)
	}

	if _, ok := topics[topic]; !ok {
		return ErrTopicMissing.WithAttribute("topic", topic)
	}

	return nil
}
