package actor

import (
	"context"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

type Actor struct {
	UserID         domain.UserID
	OrganizationID domain.OrganizationID
	Role           roles.Role
}

func (a *Actor) Can(p roles.Permission) bool {
	return a.Role.Grants(p)
}

// Authorize is Can for an application: a denial becomes a KindUnauthorized error (403) and is counted.
func (a *Actor) Authorize(ctx context.Context, p roles.Permission) error {
	if a.Can(p) {
		return nil
	}

	onDenied(ctx, a, p)

	return errors.ErrRequestUnauthorized
}
