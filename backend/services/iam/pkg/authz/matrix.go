package authz

type grant struct {
	permission Permission
	roles      []Role
}

var (
	PermissionOrganizationUpdate   = Permission{"organization", "update"}
	PermissionOrganizationDelete   = Permission{"organization", "delete"}
	PermissionOrganizationTransfer = Permission{"organization", "transfer"}
	PermissionInvitationCreate     = Permission{"invitation", "create"}
	PermissionInvitationCancel     = Permission{"invitation", "cancel"}
)

// example:begin
var (
	PermissionNoteRead   = Permission{"note", "read"}
	PermissionNoteCreate = Permission{"note", "create"}
	PermissionNoteDelete = Permission{"note", "delete"}
)

// example:end

var matrix = []grant{
	{PermissionOrganizationUpdate, []Role{RoleOwner, RoleAdmin}},
	{PermissionOrganizationDelete, []Role{RoleOwner}},
	{PermissionOrganizationTransfer, []Role{RoleOwner}},
	{PermissionInvitationCreate, []Role{RoleOwner, RoleAdmin}},
	{PermissionInvitationCancel, []Role{RoleOwner, RoleAdmin}},
	// example:begin
	{PermissionNoteRead, []Role{RoleOwner, RoleAdmin, RoleViewer}},
	{PermissionNoteCreate, []Role{RoleOwner, RoleAdmin}},
	{PermissionNoteDelete, []Role{RoleOwner}},
	// example:end
}
