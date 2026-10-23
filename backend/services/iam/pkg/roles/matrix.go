package roles

type grant struct {
	permission Permission
	roles      []Role
}

var (
	OrganizationUpdate   = Permission{"organization", "update"}
	OrganizationDelete   = Permission{"organization", "delete"}
	OrganizationTransfer = Permission{"organization", "transfer"}
)

// example:begin
var (
	NoteRead   = Permission{"note", "read"}
	NoteCreate = Permission{"note", "create"}
	NoteDelete = Permission{"note", "delete"}
)

// example:end

var matrix = []grant{
	{OrganizationUpdate, []Role{Owner, Admin}},
	{OrganizationDelete, []Role{Owner}},
	{OrganizationTransfer, []Role{Owner}},
	// example:begin
	{NoteRead, []Role{Owner, Admin, Viewer}},
	{NoteCreate, []Role{Owner, Admin}},
	{NoteDelete, []Role{Owner}},
	// example:end
}
