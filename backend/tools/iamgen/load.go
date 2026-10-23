package main

import (
	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

var ErrUnmappedRole = errors.New("role has no Better Auth membership defaults mapping")

func load() (matrix, error) {
	m := matrix{statement: convert(roles.Permissions())}

	for _, r := range roles.All() {
		defaults, err := betterAuthDefaults(r)
		if err != nil {
			return matrix{}, err
		}

		m.roles = append(m.roles, role{name: r.String(), betterAuthDefaults: defaults, permissions: convert(r.Permissions())})
	}

	return m, nil
}

var betterAuthDefaultsByRole = map[roles.Role]string{
	roles.Owner:  "ownerAc",
	roles.Admin:  "adminAc",
	roles.Viewer: "",
}

func betterAuthDefaults(r roles.Role) (string, error) {
	defaults, ok := betterAuthDefaultsByRole[r]
	if !ok {
		return "", ErrUnmappedRole.WithAttribute("role", r.String())
	}

	return defaults, nil
}

func convert(permissions []roles.Permission) []permission {
	converted := make([]permission, 0, len(permissions))

	for _, p := range permissions {
		converted = append(converted, permission{resource: p.Resource(), action: p.Action()})
	}

	return converted
}
