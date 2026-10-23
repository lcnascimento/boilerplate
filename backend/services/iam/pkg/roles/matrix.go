package roles

type grant struct {
	permission Permission
	roles      []Role
}

var (
	OrganizationTransfer = Permission{"organization", "transfer"}
	InvitationCreate     = Permission{"invitation", "create"}
	InvitationCancel     = Permission{"invitation", "cancel"}
)

// example:begin
var (
	NoteRead   = Permission{"note", "read"}
	NoteCreate = Permission{"note", "create"}
	NoteDelete = Permission{"note", "delete"}
)

// example:end

var matrix = []grant{
	{OrganizationTransfer, []Role{Owner}},
	{InvitationCreate, []Role{Owner, Admin}},
	{InvitationCancel, []Role{Owner, Admin}},
	// example:begin
	{NoteRead, []Role{Owner, Admin, Viewer}},
	{NoteCreate, []Role{Owner, Admin}},
	{NoteDelete, []Role{Owner}},
	// example:end
}
