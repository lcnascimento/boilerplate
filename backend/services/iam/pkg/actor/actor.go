package actor

import (
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
