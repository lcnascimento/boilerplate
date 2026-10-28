package eventbus

import (
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/cenkalti/backoff/v5"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/o11y/log"
)

const (
	handlerName          = "events"
	retryInitialInterval = 100 * time.Millisecond
	retryMaxInterval     = 30 * time.Second
	retryMultiplier      = 2
)

type SubscriberConfig struct {
	Brokers []string
	Group   string
	Topics  []string
	Handler Handler
}

func DLQTopic(topic, group string) string {
	return topic + "." + group + ".dlq"
}

type Subscriber struct {
	group   string
	handler Handler

	router     *message.Router
	subscriber *kafka.Subscriber
	dlq        *kafka.Publisher
	cluster    *Cluster
}

func NewSubscriber(cfg *SubscriberConfig) (*Subscriber, error) {
	if cfg == nil || len(cfg.Brokers) == 0 || cfg.Group == "" || len(cfg.Topics) == 0 || cfg.Handler == nil {
		return nil, ErrInvalidSubscriberConfig
	}

	s := &Subscriber{group: cfg.Group, handler: cfg.Handler}

	topics := slices.Clone(cfg.Topics)
	for _, topic := range cfg.Topics {
		topics = append(topics, DLQTopic(topic, s.group))
	}

	if err := checkTopics(cfg.Brokers, topics...); err != nil {
		return nil, err
	}

	if err := s.connect(cfg.Brokers); err != nil {
		return nil, err
	}

	cluster, err := DialCluster(cfg.Brokers, topics...)
	if err != nil {
		_ = s.subscriber.Close()
		_ = s.dlq.Close()

		return nil, err
	}

	s.cluster = cluster

	s.router, err = message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, ErrInvalidSubscriberConfig.WithCause(err)
	}

	for _, topic := range cfg.Topics {
		if err := s.route(topic); err != nil {
			return nil, err
		}
	}

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

func (s *Subscriber) Ping(ctx context.Context) error {
	return s.cluster.Ping(ctx)
}

func (s *Subscriber) Close() error {
	routerErr := s.router.Close()
	subscriberErr := s.subscriber.Close()
	dlqErr := s.dlq.Close()
	clusterErr := s.cluster.Close()

	for _, err := range []error{routerErr, subscriberErr, dlqErr, clusterErr} {
		if err != nil {
			return ErrSubscriberStopped.WithCause(err)
		}
	}

	return nil
}

func (s *Subscriber) route(topic string) error {
	poisonQueue, err := middleware.PoisonQueueWithFilter(dlqPublisher{s: s, topic: topic}, DLQTopic(topic, s.group), isPermanent)
	if err != nil {
		return ErrInvalidSubscriberConfig.WithCause(err)
	}

	retry := middleware.Retry{
		MaxRetries:      math.MaxInt32,
		InitialInterval: retryInitialInterval,
		MaxInterval:     retryMaxInterval,
		Multiplier:      retryMultiplier,
	}

	h := topicHandler{s: s, topic: topic}

	handler := s.router.AddConsumerHandler(topic, topic, s.subscriber, h.handle)
	handler.AddMiddleware(h.trackDelivery, poisonQueue, retry.Middleware, h.attempt, recoverer)

	return nil
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

type topicHandler struct {
	s     *Subscriber
	topic string
}

func (h topicHandler) handle(msg *message.Message) error {
	event, err := parseEvent(h.topic, msg)
	if err != nil {
		return onUndeliverable(msg.Context(), ErrPermanent.WithCause(err))
	}

	return h.s.handler.Handle(msg.Context(), &event)
}

func (h topicHandler) trackDelivery(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		msg.SetContext(withDelivery(msg.Context()))

		produced, err := next(msg)
		h.s.onConsumed(h.topic, msg, err)

		return produced, err
	}
}

func (h topicHandler) attempt(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		d := deliveryFrom(msg.Context())
		d.attempt++

		original := msg.Context()
		defer msg.SetContext(original)

		ctx, end := h.s.onAttemptStart(h.topic, msg, d.attempt)
		msg.SetContext(ctx)

		produced, err := next(msg)
		end(err)

		if isPermanent(err) {
			d.errorType = errorType(err)

			// Retry.ShouldRetry would advance the backoff on every call, doubling each interval.
			return produced, backoff.Permanent(fullReasonError{err})
		}

		return produced, err
	}
}

func recoverer(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) (produced []*message.Message, err error) {
		defer func() {
			if cause := recover(); cause != nil {
				panicErr := errors.New("%s", fmt.Sprint(cause)).WithCode("PANIC").WithKind(errors.KindCritical)
				logger.Critical(msg.Context(), panicErr, log.String("exception.stack", string(debug.Stack())))

				err = ErrPermanent.WithCause(panicErr)
			}
		}()

		return next(msg)
	}
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
	s     *Subscriber
	topic string
}

func (p dlqPublisher) Publish(topic string, messages ...*message.Message) error {
	if err := p.s.dlq.Publish(topic, messages...); err != nil {
		return err
	}

	for _, msg := range messages {
		p.s.onDeadLettered(p.topic, msg)
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
