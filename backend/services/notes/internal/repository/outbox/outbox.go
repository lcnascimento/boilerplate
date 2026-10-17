package outbox

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// ForwarderTopic names the outbox tables: watermill_notes_outbox and watermill_offsets_notes_outbox.
	ForwarderTopic = "notes_outbox"
	ConsumerGroup  = "notes-forwarder"
)

func NewPublisher(tx pgx.Tx) (message.Publisher, error) {
	publisher, err := sql.NewPublisher(
		sql.TxFromPgx(tx),
		sql.PublisherConfig{SchemaAdapter: sql.DefaultPostgreSQLSchema{}},
		watermill.NopLogger{},
	)
	if err != nil {
		return nil, err
	}

	return forwarder.NewPublisher(publisher, forwarder.PublisherConfig{ForwarderTopic: ForwarderTopic}), nil
}

// NewSubscriber reads the outbox one message at a time, in commit order, for ConsumerGroup. Its transaction
// locks the group's offset row, so of several subscribers only one consumes and the others wait.
func NewSubscriber(pool *pgxpool.Pool) (message.Subscriber, error) {
	// A deadline would redeliver a message while its handler may still be publishing it, so the next
	// message could reach Kafka first.
	noAckDeadline := time.Duration(0)

	return sql.NewSubscriber(
		sql.BeginnerFromPgx(pool),
		sql.SubscriberConfig{
			ConsumerGroup:    ConsumerGroup,
			AckDeadline:      &noAckDeadline,
			SchemaAdapter:    sql.DefaultPostgreSQLSchema{SubscribeBatchSize: 1},
			OffsetsAdapter:   sql.DefaultPostgreSQLOffsetsAdapter{},
			InitializeSchema: false,
		},
		watermill.NopLogger{},
	)
}
