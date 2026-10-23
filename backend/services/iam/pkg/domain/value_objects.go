package domain

import (
	"github.com/google/uuid"

	"github.com/lcnascimento/go-kit/util"
)

const (
	userIDPrefix         string = "user"
	organizationIDPrefix string = "org"
	memberIDPrefix       string = "member"
	invitationIDPrefix   string = "invitation"
	sessionIDPrefix      string = "session"
)

type UserID struct {
	util.CanonicalID
}

func ParseUserID(s string) (UserID, error) {
	id, err := util.ParseCanonicalID(userIDPrefix, s)
	if err != nil {
		return UserID{}, ErrInvalidUserID.WithCause(err)
	}

	return UserID{id}, nil
}

func UserIDFromUUID(u uuid.UUID) UserID {
	return UserID{util.CanonicalIDFromUUID(userIDPrefix, u)}
}

type OrganizationID struct {
	util.CanonicalID
}

func ParseOrganizationID(s string) (OrganizationID, error) {
	id, err := util.ParseCanonicalID(organizationIDPrefix, s)
	if err != nil {
		return OrganizationID{}, ErrInvalidOrganizationID.WithCause(err)
	}

	return OrganizationID{id}, nil
}

func MustParseOrganizationID(s string) OrganizationID {
	id, err := ParseOrganizationID(s)
	if err != nil {
		panic(err)
	}

	return id
}

func OrganizationIDFromUUID(u uuid.UUID) OrganizationID {
	return OrganizationID{util.CanonicalIDFromUUID(organizationIDPrefix, u)}
}

type MemberID struct {
	util.CanonicalID
}

func MemberIDFromUUID(u uuid.UUID) MemberID {
	return MemberID{util.CanonicalIDFromUUID(memberIDPrefix, u)}
}

type InvitationID struct {
	util.CanonicalID
}

func InvitationIDFromUUID(u uuid.UUID) InvitationID {
	return InvitationID{util.CanonicalIDFromUUID(invitationIDPrefix, u)}
}

type SessionID struct {
	util.CanonicalID
}

func SessionIDFromUUID(u uuid.UUID) SessionID {
	return SessionID{util.CanonicalIDFromUUID(sessionIDPrefix, u)}
}
