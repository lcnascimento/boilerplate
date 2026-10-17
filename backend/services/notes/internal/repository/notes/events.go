package notes

import (
	"context"
	"maps"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/outbox"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const source = "notes"

var ErrEmit = errors.New("failed to write the Integration Event to the outbox").WithCode("OUTBOX_EMIT")

func (r *Repository) emit(ctx context.Context, tx pgx.Tx, id domain.NoteID, at time.Time, event proto.Message) (err error) {
	msg, err := r.newMessage(id, at, event)
	if err != nil {
		return ErrEmit.WithCause(err)
	}

	ctx, span := onEmitStart(ctx, msg)
	defer func() { onEmitEnd(span, err) }()

	msg.SetContext(ctx)

	publisher, err := outbox.NewPublisher(tx)
	if err != nil {
		return ErrEmit.WithCause(err)
	}

	if err := publisher.Publish(events.Topic, msg); err != nil {
		return ErrEmit.WithCause(err)
	}

	return nil
}

func (r *Repository) newMessage(id domain.NoteID, at time.Time, event proto.Message) (*message.Message, error) {
	value, err := r.schemaIDs.Encode(event)
	if err != nil {
		return nil, err
	}

	messageID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	envelope := events.Envelope{
		ID:              messageID.String(),
		Source:          source,
		Type:            string(event.ProtoReflect().Descriptor().FullName()),
		Time:            at.UTC(),
		Subject:         id.String(),
		PartitionKey:    id.String(),
		SpecVersion:     events.SpecVersion,
		DataContentType: events.ContentType,
	}

	msg := message.NewMessage(envelope.ID, value)
	maps.Copy(msg.Metadata, envelope.Headers())

	return msg, nil
}
