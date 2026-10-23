package authz

type Permission struct {
	resource string
	action   string
}

func (p Permission) Resource() string {
	return p.resource
}

func (p Permission) Action() string {
	return p.action
}

func (p Permission) String() string {
	return p.resource + ":" + p.action
}

func Permissions() []Permission {
	all := make([]Permission, 0, len(matrix))

	for _, g := range matrix {
		all = append(all, g.permission)
	}

	return all
}
