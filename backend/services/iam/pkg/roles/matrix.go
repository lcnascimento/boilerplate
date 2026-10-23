package roles

type grant struct {
	permission Permission
	roles      []Role
}

// example:begin
var (
	NoteRead   = Permission{"note", "read"}
	NoteCreate = Permission{"note", "create"}
	NoteDelete = Permission{"note", "delete"}
)

// example:end

var matrix = []grant{
	// example:begin
	{NoteRead, []Role{Owner, Admin, Viewer}},
	{NoteCreate, []Role{Owner, Admin}},
	{NoteDelete, []Role{Owner}},
	// example:end
}
