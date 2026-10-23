package idomain

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

var ErrUntranslatableChange = errors.New("the identity change has no Integration Event").
	WithCode("UNTRANSLATABLE_CHANGE")

type Change struct {
	ID            int64
	TransactionID string
	OccurredAt    time.Time
	Table         string
	Operation     string
	Fields        map[string]string

	ActorID             uuid.NullUUID
	ActorOrganizationID uuid.NullUUID
	IPHash              string
}

type Event struct {
	Topic         string
	Subject       string
	Time          time.Time
	CorrelationID string
	Message       proto.Message

	ActorID             string
	ActorOrganizationID string
	IPHash              string
}

type changeKind struct {
	table     string
	operation string
}

type userEvent struct {
	userIDField string
	message     func(userID string, at *timestamppb.Timestamp) proto.Message
}

var userEvents = map[changeKind]userEvent{
	{table: "user", operation: "INSERT"}: {
		userIDField: "id",
		message: func(userID string, at *timestamppb.Timestamp) proto.Message {
			return &events.UserCreated{UserId: userID, CreatedAt: at}
		},
	},
	{table: "user", operation: "DELETE"}: {
		userIDField: "id",
		message: func(userID string, at *timestamppb.Timestamp) proto.Message {
			return &events.UserDeleted{UserId: userID, DeletedAt: at}
		},
	},
	{table: "account", operation: "UPDATE"}: {
		userIDField: "userId",
		message: func(userID string, at *timestamppb.Timestamp) proto.Message {
			return &events.PasswordChanged{UserId: userID, ChangedAt: at}
		},
	},
}

func (c *Change) Event() (*Event, error) {
	translation, ok := userEvents[changeKind{table: c.Table, operation: c.Operation}]
	if !ok {
		return nil, c.untranslatable()
	}

	raw, err := uuid.Parse(c.Fields[translation.userIDField])
	if err != nil {
		return nil, c.untranslatable().WithCause(err)
	}

	userID := domain.UserIDFromUUID(raw).String()

	return &Event{
		Topic:         events.UserTopic,
		Subject:       userID,
		Time:          c.OccurredAt,
		CorrelationID: c.TransactionID,
		Message:       translation.message(userID, timestamppb.New(c.OccurredAt)),

		ActorID:             prefixed(c.ActorID, domain.UserIDFromUUID),
		ActorOrganizationID: prefixed(c.ActorOrganizationID, domain.OrganizationIDFromUUID),
		IPHash:              c.IPHash,
	}, nil
}

func prefixed[ID fmt.Stringer](id uuid.NullUUID, fromUUID func(uuid.UUID) ID) string {
	if !id.Valid {
		return ""
	}

	return fromUUID(id.UUID).String()
}

func (c *Change) untranslatable() errors.CustomError {
	return ErrUntranslatableChange.
		WithAttribute("change_id", strconv.FormatInt(c.ID, 10)).
		WithAttribute("table", c.Table).
		WithAttribute("operation", c.Operation)
}
