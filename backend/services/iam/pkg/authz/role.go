package authz

import "slices"

type Role struct {
	name string
}

var (
	RoleOwner  = Role{"owner"}
	RoleAdmin  = Role{"admin"}
	RoleViewer = Role{"viewer"}
)

func AllRoles() []Role {
	return []Role{RoleOwner, RoleAdmin, RoleViewer}
}

func ParseRole(s string) (Role, error) {
	for _, r := range AllRoles() {
		if r.name == s {
			return r, nil
		}
	}

	return Role{}, ErrUnknownRole
}

func (r Role) String() string {
	return r.name
}

func (r Role) Grants(p Permission) bool {
	for _, g := range matrix {
		if g.permission == p {
			return slices.Contains(g.roles, r)
		}
	}

	return false
}

func (r Role) Permissions() []Permission {
	var granted []Permission

	for _, g := range matrix {
		if slices.Contains(g.roles, r) {
			granted = append(granted, g.permission)
		}
	}

	return granted
}
