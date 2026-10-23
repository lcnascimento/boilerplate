package consumer

import (
	"context"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	iamevents "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
)

const Group = "notes"

var Topics = []string{iamevents.OrganizationTopic}

type Handler struct {
	notes transport.OrganizationNotesService
}

func New(notes transport.OrganizationNotesService, brokers []string, opts ...eventbus.ConsumerOption) *eventbus.Consumer {
	return eventbus.NewConsumer(&eventbus.SubscriberConfig{
		Brokers: brokers,
		Group:   Group,
		Topics:  Topics,
		Handler: NewHandler(notes),
	}, opts...)
}

func NewHandler(notes transport.OrganizationNotesService) *Handler {
	return &Handler{notes: notes}
}

func (h *Handler) Handle(ctx context.Context, event *eventbus.Event) error {
	message, _ := event.Message()

	deleted, ok := message.(*iamevents.OrganizationDeleted)
	if !ok {
		return nil
	}

	orgID, err := iamdomain.ParseOrganizationID(deleted.GetOrganizationId())
	if err != nil {
		return eventbus.ErrPermanent.WithCause(err)
	}

	return h.notes.DeleteOrganizationNotes(ctx, orgID)
}
