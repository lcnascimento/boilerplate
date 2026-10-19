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
	// ForwarderTopic is what watermill-sql derives the table names watermill_notes_outbox and watermill_offsets_notes_outbox from.
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

// NewSubscriber relies on watermill-sql locking the group's offset row in each transaction, so only one replica consumes and commit order holds.
func NewSubscriber(pool *pgxpool.Pool) (message.Subscriber, error) {
	// watermill-sql redelivers a message whose ack deadline passes while its handler may still be publishing it.
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
