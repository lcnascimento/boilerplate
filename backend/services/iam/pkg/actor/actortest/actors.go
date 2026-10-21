package actortest

import (
	"context"

	"github.com/google/uuid"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/internal/ctxkey"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

func Context(ctx context.Context, a *actor.Actor) context.Context {
	return context.WithValue(ctx, ctxkey.Actor{}, a)
}

func Owner(orgID domain.OrganizationID) *actor.Actor {
	return newActor(orgID, roles.Owner)
}

func Admin(orgID domain.OrganizationID) *actor.Actor {
	return newActor(orgID, roles.Admin)
}

func Viewer(orgID domain.OrganizationID) *actor.Actor {
	return newActor(orgID, roles.Viewer)
}

func newActor(orgID domain.OrganizationID, role roles.Role) *actor.Actor {
	userID, err := domain.ParseUserID("user_" + uuid.NewString())
	if err != nil {
		panic(err)
	}

	return &actor.Actor{UserID: userID, OrganizationID: orgID, Role: role}
}
