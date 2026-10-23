package actor

import (
	"context"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/authz"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
)

type Actor struct {
	UserID         domain.UserID
	OrganizationID domain.OrganizationID
	Role           authz.Role
}

func (a *Actor) Can(p authz.Permission) bool {
	return a.Role.Grants(p)
}

func (a *Actor) Authorize(ctx context.Context, p authz.Permission) error {
	if a.Can(p) {
		return nil
	}

	onDenied(ctx, a, p)

	return errors.ErrRequestUnauthorized
}
