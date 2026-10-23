package roles

import "slices"

type Role struct {
	name string
}

var (
	Owner  = Role{"owner"}
	Admin  = Role{"admin"}
	Viewer = Role{"viewer"}
)

func All() []Role {
	return []Role{Owner, Admin, Viewer}
}

func ParseRole(s string) (Role, error) {
	for _, r := range All() {
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
