package events

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/cenkalti/backoff/v5"

	"github.com/lcnascimento/go-kit/errors"
)

const (
	handlerName          = "events"
	retryInitialInterval = 100 * time.Millisecond
	retryMaxInterval     = 30 * time.Second
	retryMultiplier      = 2
)

type SubscriberConfig struct {
	Brokers  []string
	Group    string
	Observer Observer
}

func DLQTopic(topic, group string) string {
	return topic + "." + group + ".dlq"
}

type Subscriber struct {
	topic    string
	group    string
	observer Observer

	router     *message.Router
	subscriber *kafka.Subscriber
	dlq        *kafka.Publisher
}

func NewSubscriber(cfg SubscriberConfig) (*Subscriber, error) {
	return newSubscriber(cfg, Topic)
}

func newSubscriber(cfg SubscriberConfig, topic string) (*Subscriber, error) {
	if len(cfg.Brokers) == 0 || cfg.Group == "" || cfg.Observer == nil {
		return nil, ErrInvalidSubscriberConfig
	}

	s := &Subscriber{topic: topic, group: cfg.Group, observer: cfg.Observer}

	if err := checkTopics(cfg.Brokers, s.topic, s.dlqTopic()); err != nil {
		return nil, err
	}

	if err := s.connect(cfg.Brokers); err != nil {
		return nil, err
	}

	poisonQueue, err := middleware.PoisonQueueWithFilter(dlqPublisher{s}, s.dlqTopic(), isPermanent)
	if err != nil {
		return nil, ErrInvalidSubscriberConfig.WithCause(err)
	}

	s.router, err = message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, ErrInvalidSubscriberConfig.WithCause(err)
	}

	retry := middleware.Retry{
		MaxRetries:      math.MaxInt32,
		InitialInterval: retryInitialInterval,
		MaxInterval:     retryMaxInterval,
		Multiplier:      retryMultiplier,
	}

	s.router.AddMiddleware(s.trackDelivery, poisonQueue, retry.Middleware, s.attempt, middleware.Recoverer)
	s.router.AddConsumerHandler(handlerName, s.topic, s.subscriber, s.handle)

	return s, nil
}

func (s *Subscriber) Run(ctx context.Context) error {
	if err := s.router.Run(ctx); err != nil {
		return ErrSubscriberStopped.WithCause(err)
	}

	return nil
}

func (s *Subscriber) Running() <-chan struct{} {
	return s.router.Running()
}

func (s *Subscriber) Close() error {
	routerErr := s.router.Close()
	subscriberErr := s.subscriber.Close()
	dlqErr := s.dlq.Close()

	for _, err := range []error{routerErr, subscriberErr, dlqErr} {
		if err != nil {
			return ErrSubscriberStopped.WithCause(err)
		}
	}

	return nil
}

func (s *Subscriber) dlqTopic() string {
	return DLQTopic(s.topic, s.group)
}

func (s *Subscriber) connect(brokers []string) error {
	consumerConfig := kafka.DefaultSaramaSubscriberConfig()
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	subscriber, err := kafka.NewSubscriber(kafka.SubscriberConfig{
		Brokers:               brokers,
		ConsumerGroup:         s.group,
		OverwriteSaramaConfig: consumerConfig,
	}, watermill.NopLogger{})
	if err != nil {
		return ErrKafkaUnavailable.WithCause(err)
	}

	producerConfig := kafka.DefaultSaramaSyncPublisherConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Idempotent = true
	producerConfig.Net.MaxOpenRequests = 1

	dlq, err := kafka.NewPublisher(kafka.PublisherConfig{
		Brokers:               brokers,
		Marshaler:             kafka.NewWithPartitioningMarshaler(originalKey),
		OverwriteSaramaConfig: producerConfig,
	}, watermill.NopLogger{})
	if err != nil {
		_ = subscriber.Close()

		return ErrKafkaUnavailable.WithCause(err)
	}

	s.subscriber = subscriber
	s.dlq = dlq

	return nil
}

func checkTopics(brokers []string, topics ...string) error {
	admin, err := sarama.NewClusterAdmin(brokers, sarama.NewConfig())
	if err != nil {
		return ErrKafkaUnavailable.WithCause(err)
	}

	defer func() { _ = admin.Close() }()

	existing, err := admin.ListTopics()
	if err != nil {
		return ErrKafkaUnavailable.WithCause(err)
	}

	for _, topic := range topics {
		if _, ok := existing[topic]; !ok {
			return ErrTopicMissing.WithAttribute("topic", topic)
		}
	}

	return nil
}

func originalKey(_ string, msg *message.Message) (string, error) {
	key, _ := kafka.MessageKeyFromCtx(msg.Context())

	return string(key), nil
}

func (s *Subscriber) handle(msg *message.Message) error {
	event, err := parseEvent(s.topic, msg)
	if err != nil {
		return ErrPermanent.WithCause(err)
	}

	return notify(msg.Context(), s.observer, &event)
}

func (s *Subscriber) trackDelivery(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		msg.SetContext(withDelivery(msg.Context()))

		produced, err := h(msg)
		s.onConsumed(msg, err)

		return produced, err
	}
}

func (s *Subscriber) attempt(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		d := deliveryFrom(msg.Context())
		d.attempt++

		original := msg.Context()
		defer msg.SetContext(original)

		ctx, end := s.onAttemptStart(msg, d.attempt)
		msg.SetContext(ctx)

		produced, err := h(msg)
		err = classify(err)
		end(err)

		if isPermanent(err) {
			d.errorType = errorType(err)

			// Retry.ShouldRetry would advance the backoff on every call, doubling each interval.
			return produced, backoff.Permanent(fullReasonError{err})
		}

		return produced, err
	}
}

func classify(err error) error {
	var panicked middleware.RecoveredPanicError
	if errors.As(err, &panicked) {
		return ErrPermanent.WithCause(ErrHandlerPanicked.WithCause(err))
	}

	return err
}

type fullReasonError struct {
	error
}

func (e fullReasonError) Error() string {
	return strings.Join(append([]string{e.error.Error()}, errors.Reasons(e.error)...), ": ")
}

func (e fullReasonError) Unwrap() error {
	return e.error
}

func isPermanent(err error) bool {
	return errors.Is(err, ErrPermanent)
}

type dlqPublisher struct {
	s *Subscriber
}

func (p dlqPublisher) Publish(topic string, messages ...*message.Message) error {
	if err := p.s.dlq.Publish(topic, messages...); err != nil {
		return err
	}

	for _, msg := range messages {
		p.s.onDeadLettered(msg)
	}

	return nil
}

func (dlqPublisher) Close() error {
	return nil
}

type delivery struct {
	attempt   int
	errorType string
}

type deliveryKey struct{}

func withDelivery(ctx context.Context) context.Context {
	return context.WithValue(ctx, deliveryKey{}, &delivery{})
}

func deliveryFrom(ctx context.Context) *delivery {
	if d, ok := ctx.Value(deliveryKey{}).(*delivery); ok {
		return d
	}

	return &delivery{}
}
