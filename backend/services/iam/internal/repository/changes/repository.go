package changes

import (
	"context"
	"encoding/json"
	"maps"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/outbox"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/changes/internal/db"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
)

const source = "iam"

var Outbox = outbox.MustNew("iam")

var (
	ErrQuery = errors.New("failed to query the identity changes")
	ErrEmit  = errors.New("failed to write the Integration Event to the outbox").WithCode("OUTBOX_EMIT")
)

type Repository struct {
	pool      *pgxpool.Pool
	schemaIDs eventbus.SchemaIDs
}

func New(pool *pgxpool.Pool, schemaIDs eventbus.SchemaIDs) *Repository {
	return &Repository{pool: pool, schemaIDs: schemaIDs}
}

func (r *Repository) Translate(ctx context.Context, limit int32) ([]*idomain.Event, error) {
	var translated []*idomain.Event

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)

		rows, err := queries.LockChanges(ctx, limit)
		if err != nil {
			return ErrQuery.WithCause(err)
		}

		if len(rows) == 0 {
			return nil
		}

		ids := make([]int64, 0, len(rows))
		translated = make([]*idomain.Event, 0, len(rows))

		for _, row := range rows {
			event, err := r.emit(ctx, tx, &row)
			if err != nil {
				return err
			}

			ids = append(ids, row.ID)
			translated = append(translated, event)
		}

		if err := queries.DeleteChanges(ctx, ids); err != nil {
			return ErrQuery.WithCause(err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return translated, nil
}

func (r *Repository) OldestPendingAge(ctx context.Context) (time.Duration, error) {
	seconds, err := db.New(r.pool).OldestChangeAge(ctx)
	if err != nil {
		return 0, ErrQuery.WithCause(err)
	}

	return time.Duration(seconds * float64(time.Second)), nil
}

func (r *Repository) emit(ctx context.Context, tx pgx.Tx, row *db.LockChangesRow) (*idomain.Event, error) {
	change, err := toChange(row)
	if err != nil {
		return nil, err
	}

	event, err := change.Event()
	if err != nil {
		return nil, err
	}

	msg, err := r.newMessage(event)
	if err != nil {
		return nil, ErrEmit.WithCause(err)
	}

	if err := Outbox.Publish(ctx, tx, event.Topic, msg); err != nil {
		return nil, ErrEmit.WithCause(err)
	}

	return event, nil
}

func toChange(row *db.LockChangesRow) (*idomain.Change, error) {
	change := &idomain.Change{
		ID:            row.ID,
		TransactionID: row.TransactionID,
		OccurredAt:    row.OccurredAt,
		Table:         row.TableName,
		Operation:     row.Operation,
	}

	if err := json.Unmarshal(row.Fields, &change.Fields); err != nil {
		return nil, idomain.ErrUntranslatableChange.WithCause(err)
	}

	return change, nil
}

func (r *Repository) newMessage(event *idomain.Event) (*message.Message, error) {
	value, err := r.schemaIDs.Encode(event.Message)
	if err != nil {
		return nil, err
	}

	messageID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	envelope := eventbus.Envelope{
		ID:              messageID.String(),
		Source:          source,
		Type:            event.Type(),
		Time:            event.Time.UTC(),
		Subject:         event.Subject,
		PartitionKey:    event.Subject,
		SpecVersion:     eventbus.SpecVersion,
		DataContentType: eventbus.ContentType,
		CorrelationID:   event.CorrelationID,
	}

	msg := message.NewMessage(envelope.ID, value)
	maps.Copy(msg.Metadata, envelope.Headers())

	return msg, nil
}
