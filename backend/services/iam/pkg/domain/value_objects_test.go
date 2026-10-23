package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
)

func TestUserID_ParseAcceptsUserPrefixedUUID(t *testing.T) {
	t.Parallel()

	id, err := domain.ParseUserID("user_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d")

	require.NoError(t, err)
	require.Equal(t, "user_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", id.String())
}

func TestUserID_ParseRejectsMalformedIDs(t *testing.T) {
	t.Parallel()

	inputs := map[string]string{
		"empty":        "",
		"wrong prefix": "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		"no prefix":    "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		"bad uuid":     "user_not-a-uuid",
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.ParseUserID(input)

			require.ErrorIs(t, err, domain.ErrInvalidUserID)
		})
	}
}

func TestOrganizationID_ParseAcceptsOrgPrefixedUUID(t *testing.T) {
	t.Parallel()

	id, err := domain.ParseOrganizationID("org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d")

	require.NoError(t, err)
	require.Equal(t, "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", id.String())
}

func TestOrganizationID_ParseRejectsMalformedIDs(t *testing.T) {
	t.Parallel()

	inputs := map[string]string{
		"empty":        "",
		"wrong prefix": "user_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		"no prefix":    "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		"bad uuid":     "org_not-a-uuid",
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.ParseOrganizationID(input)

			require.ErrorIs(t, err, domain.ErrInvalidOrganizationID)
		})
	}
}

func TestOrganizationID_FromUUIDAddsTheOrgPrefix(t *testing.T) {
	t.Parallel()

	id := domain.OrganizationIDFromUUID(uuid.MustParse("3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"))

	require.Equal(t, "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", id.String())
}

func TestUserID_FromUUIDAddsTheUserPrefix(t *testing.T) {
	t.Parallel()

	id := domain.UserIDFromUUID(uuid.MustParse("3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"))

	require.Equal(t, "user_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", id.String())
}

func TestMemberID_FromUUIDAddsTheMemberPrefix(t *testing.T) {
	t.Parallel()

	id := domain.MemberIDFromUUID(uuid.MustParse("3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"))

	require.Equal(t, "member_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", id.String())
}

func TestInvitationID_FromUUIDAddsTheInvitationPrefix(t *testing.T) {
	t.Parallel()

	id := domain.InvitationIDFromUUID(uuid.MustParse("3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"))

	require.Equal(t, "invitation_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", id.String())
}

func TestOrganizationID_MustParsePanicsOnMalformedID(t *testing.T) {
	t.Parallel()

	require.Equal(t, "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		domain.MustParseOrganizationID("org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d").String())
	require.Panics(t, func() { domain.MustParseOrganizationID("user_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d") })
}
