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

type translation func(c *Change) (*Event, error)

var translations = map[changeKind]translation{
	{table: "user", operation: "INSERT"}: userEvent("id", func(userID string, at *timestamppb.Timestamp) proto.Message {
		return &events.UserCreated{UserId: userID, CreatedAt: at}
	}),
	{table: "user", operation: "DELETE"}: userEvent("id", func(userID string, at *timestamppb.Timestamp) proto.Message {
		return &events.UserDeleted{UserId: userID, DeletedAt: at}
	}),
	{table: "account", operation: "UPDATE"}: userEvent("userId", func(userID string, at *timestamppb.Timestamp) proto.Message {
		return &events.PasswordChanged{UserId: userID, ChangedAt: at}
	}),
	{table: "session", operation: "INSERT"}: sessionEvent(func(sessionID, userID string, at *timestamppb.Timestamp) proto.Message {
		return &events.SessionStarted{SessionId: sessionID, UserId: userID, StartedAt: at}
	}),
	{table: "session", operation: "DELETE"}: sessionEvent(func(sessionID, userID string, at *timestamppb.Timestamp) proto.Message {
		return &events.SessionEnded{SessionId: sessionID, UserId: userID, EndedAt: at}
	}),
	{table: "sign_in", operation: "FAILED"}: signInFailed,
}

var signInFailureReasons = map[string]events.SignInFailed_Reason{
	"invalid_credentials": events.SignInFailed_REASON_INVALID_CREDENTIALS,
	"invalid_input":       events.SignInFailed_REASON_INVALID_INPUT,
}

func (c *Change) Event() (*Event, error) {
	translate, ok := translations[changeKind{table: c.Table, operation: c.Operation}]
	if !ok {
		return nil, c.untranslatable()
	}

	return translate(c)
}

func userEvent(userIDField string, message func(userID string, at *timestamppb.Timestamp) proto.Message) translation {
	return func(c *Change) (*Event, error) {
		userID, err := fieldID(c, userIDField, domain.UserIDFromUUID)
		if err != nil {
			return nil, err
		}

		return c.event(events.UserTopic, userID, message(userID, timestamppb.New(c.OccurredAt))), nil
	}
}

func sessionEvent(message func(sessionID, userID string, at *timestamppb.Timestamp) proto.Message) translation {
	return func(c *Change) (*Event, error) {
		sessionID, err := fieldID(c, "id", domain.SessionIDFromUUID)
		if err != nil {
			return nil, err
		}

		userID, err := fieldID(c, "userId", domain.UserIDFromUUID)
		if err != nil {
			return nil, err
		}

		event := c.event(events.SessionTopic, sessionID, message(sessionID, userID, timestamppb.New(c.OccurredAt)))
		if c.ActorID.Valid {
			return event, nil
		}

		event.ActorID = userID
		event.ActorOrganizationID = ""

		if c.Fields["activeOrganizationId"] != "" {
			event.ActorOrganizationID, err = fieldID(c, "activeOrganizationId", domain.OrganizationIDFromUUID)
			if err != nil {
				return nil, err
			}
		}

		return event, nil
	}
}

func signInFailed(c *Change) (*Event, error) {
	reason, ok := signInFailureReasons[c.Fields["reason"]]
	if !ok {
		return nil, c.untranslatable().WithAttribute("reason", c.Fields["reason"])
	}

	return c.event(events.SessionTopic, "", &events.SignInFailed{
		Reason:   reason,
		FailedAt: timestamppb.New(c.OccurredAt),
	}), nil
}

func (c *Change) event(topic, subject string, message proto.Message) *Event {
	return &Event{
		Topic:         topic,
		Subject:       subject,
		Time:          c.OccurredAt,
		CorrelationID: c.TransactionID,
		Message:       message,

		ActorID:             prefixed(c.ActorID, domain.UserIDFromUUID),
		ActorOrganizationID: prefixed(c.ActorOrganizationID, domain.OrganizationIDFromUUID),
		IPHash:              c.IPHash,
	}
}

func fieldID[ID fmt.Stringer](c *Change, field string, fromUUID func(uuid.UUID) ID) (string, error) {
	raw, err := uuid.Parse(c.Fields[field])
	if err != nil {
		return "", c.untranslatable().WithAttribute("field", field).WithCause(err)
	}

	return fromUUID(raw).String(), nil
}

func (e *Event) Type() string {
	return string(e.Message.ProtoReflect().Descriptor().FullName())
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
