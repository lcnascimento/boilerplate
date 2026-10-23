package actortest

import (
	"context"

	"github.com/google/uuid"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/internal/ctxkey"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/authz"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
)

func Context(ctx context.Context, a *actor.Actor) context.Context {
	return context.WithValue(ctx, ctxkey.Actor{}, a)
}

func Owner(orgID domain.OrganizationID) *actor.Actor {
	return newActor(orgID, authz.RoleOwner)
}

func Admin(orgID domain.OrganizationID) *actor.Actor {
	return newActor(orgID, authz.RoleAdmin)
}

func Viewer(orgID domain.OrganizationID) *actor.Actor {
	return newActor(orgID, authz.RoleViewer)
}

func newActor(orgID domain.OrganizationID, role authz.Role) *actor.Actor {
	userID, err := domain.ParseUserID("user_" + uuid.NewString())
	if err != nil {
		panic(err)
	}

	return &actor.Actor{UserID: userID, OrganizationID: orgID, Role: role}
}
