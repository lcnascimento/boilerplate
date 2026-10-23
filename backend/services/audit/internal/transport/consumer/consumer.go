package consumer

import (
	"context"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport"
)

const Group = "audit"

type Handler struct {
	audit transport.AuditService
}

func New(audit transport.AuditService, brokers []string, opts ...eventbus.ConsumerOption) *eventbus.Consumer {
	return eventbus.NewConsumer(&eventbus.SubscriberConfig{
		Brokers: brokers,
		Group:   Group,
		Topics:  Topics,
		Handler: NewHandler(audit),
	}, opts...)
}

func NewHandler(audit transport.AuditService) *Handler {
	return &Handler{audit: audit}
}

func (h *Handler) Handle(ctx context.Context, event *eventbus.Event) error {
	return h.audit.Record(ctx, &idomain.Entry{
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

		ActorID:             event.ActorID,
		ActorOrganizationID: event.ActorOrganizationID,
		IPHash:              event.IPHash,
		CorrelationID:       event.CorrelationID,
	})
}
