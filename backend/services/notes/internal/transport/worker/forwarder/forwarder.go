package forwarder

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lcnascimento/go-kit/outbox"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

func New(
	ctx context.Context, o outbox.Outbox, pool *pgxpool.Pool, brokers []string, shutdown func(),
) (*outbox.Forwarder, error) {
	return outbox.NewForwarder(ctx, o, pool,
		outbox.KafkaTarget{Brokers: brokers, Topics: []string{events.Topic}},
		outbox.WithShutdown(shutdown),
		outbox.WithValidation(validEnvelope),
	)
}

func validEnvelope(msg *message.Message) error {
	_, err := eventbus.ParseEnvelope(msg.Metadata)

	return err
}
