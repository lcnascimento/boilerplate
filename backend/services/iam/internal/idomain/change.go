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

type translation struct {
	topic     string
	translate func(f *fields, at *timestamppb.Timestamp) (subject string, message proto.Message)
}

var translations = map[changeKind]translation{
	{table: "user", operation: "INSERT"}: {
		topic: events.UserTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			userID := f.userID("id")

			return userID, &events.UserCreated{UserId: userID, CreatedAt: at}
		},
	},
	{table: "user", operation: "DELETE"}: {
		topic: events.UserTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			userID := f.userID("id")

			return userID, &events.UserDeleted{UserId: userID, DeletedAt: at}
		},
	},
	{table: "account", operation: "UPDATE"}: {
		topic: events.UserTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			userID := f.userID("userId")

			return userID, &events.PasswordChanged{UserId: userID, ChangedAt: at}
		},
	},
	{table: "organization", operation: "INSERT"}: {
		topic: events.OrganizationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			organizationID := f.organizationID("id")

			return organizationID, &events.OrganizationCreated{OrganizationId: organizationID, CreatedAt: at}
		},
	},
	{table: "organization", operation: "UPDATE"}: {
		topic: events.OrganizationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			organizationID := f.organizationID("id")

			return organizationID, &events.OrganizationRenamed{OrganizationId: organizationID, RenamedAt: at}
		},
	},
	{table: "organization", operation: "DELETE"}: {
		topic: events.OrganizationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			organizationID := f.organizationID("id")

			return organizationID, &events.OrganizationDeleted{OrganizationId: organizationID, DeletedAt: at}
		},
	},
	{table: "member", operation: "INSERT"}: {
		topic: events.OrganizationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			organizationID := f.organizationID("organizationId")

			return organizationID, &events.MemberAdded{
				MemberId:       f.memberID("id"),
				OrganizationId: organizationID,
				UserId:         f.userID("userId"),
				Role:           f.values["role"],
				AddedAt:        at,
			}
		},
	},
	{table: "member", operation: "UPDATE"}: {
		topic: events.OrganizationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			organizationID := f.organizationID("organizationId")

			return organizationID, &events.MemberRoleChanged{
				MemberId:       f.memberID("id"),
				OrganizationId: organizationID,
				UserId:         f.userID("userId"),
				Role:           f.values["role"],
				ChangedAt:      at,
			}
		},
	},
	{table: "member", operation: "DELETE"}: {
		topic: events.OrganizationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			organizationID := f.organizationID("organizationId")

			return organizationID, &events.MemberRemoved{
				MemberId:       f.memberID("id"),
				OrganizationId: organizationID,
				UserId:         f.userID("userId"),
				RemovedAt:      at,
			}
		},
	},
	{table: "invitation", operation: "INSERT"}: {
		topic: events.InvitationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			invitationID := f.invitationID("id")

			return invitationID, &events.InvitationCreated{
				InvitationId:   invitationID,
				OrganizationId: f.organizationID("organizationId"),
				InviterId:      f.userID("inviterId"),
				Role:           f.values["role"],
				CreatedAt:      at,
			}
		},
	},
	{table: "invitation", operation: "UPDATE"}: {
		topic: events.InvitationTopic,
		translate: func(f *fields, at *timestamppb.Timestamp) (string, proto.Message) {
			invitationID, organizationID := f.invitationID("id"), f.organizationID("organizationId")

			switch status := f.values["status"]; status {
			case "accepted":
				return invitationID, &events.InvitationAccepted{
					InvitationId: invitationID, OrganizationId: organizationID, AcceptedAt: at,
				}
			case "rejected":
				return invitationID, &events.InvitationDeclined{
					InvitationId: invitationID, OrganizationId: organizationID, DeclinedAt: at,
				}
			case "canceled":
				return invitationID, &events.InvitationCanceled{
					InvitationId: invitationID, OrganizationId: organizationID, CanceledAt: at,
				}
			default:
				f.fail(errUnknownInvitationStatus.WithAttribute("status", status))

				return invitationID, nil
			}
		},
	},
}

var errUnknownInvitationStatus = errors.New("the invitation status has no Integration Event")

type fields struct {
	values map[string]string
	err    error
}

func (f *fields) fail(err error) {
	if f.err == nil {
		f.err = err
	}
}

func (f *fields) uuid(name string) uuid.UUID {
	id, err := uuid.Parse(f.values[name])
	if err != nil {
		f.fail(err)
	}

	return id
}

func (f *fields) userID(name string) string {
	return domain.UserIDFromUUID(f.uuid(name)).String()
}

func (f *fields) organizationID(name string) string {
	return domain.OrganizationIDFromUUID(f.uuid(name)).String()
}

func (f *fields) memberID(name string) string {
	return domain.MemberIDFromUUID(f.uuid(name)).String()
}

func (f *fields) invitationID(name string) string {
	return domain.InvitationIDFromUUID(f.uuid(name)).String()
}

func (c *Change) Event() (*Event, error) {
	translation, ok := translations[changeKind{table: c.Table, operation: c.Operation}]
	if !ok {
		return nil, c.untranslatable()
	}

	f := &fields{values: c.Fields}

	subject, message := translation.translate(f, timestamppb.New(c.OccurredAt))
	if f.err != nil {
		return nil, c.untranslatable().WithCause(f.err)
	}

	return &Event{
		Topic:         translation.topic,
		Subject:       subject,
		Time:          c.OccurredAt,
		CorrelationID: c.TransactionID,
		Message:       message,

		ActorID:             prefixed(c.ActorID, domain.UserIDFromUUID),
		ActorOrganizationID: prefixed(c.ActorOrganizationID, domain.OrganizationIDFromUUID),
		IPHash:              c.IPHash,
	}, nil
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
