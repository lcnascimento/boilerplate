package domain

import "time"

// NoteContent is what a client writes; models.CreateNoteRequest embeds it, so
// its validate tags are the only place the rules live.
type NoteContent struct {
	Title string `json:"title" validate:"required,max=120"`
	Body  string `json:"body"  validate:"max=10000"`
}

// Note is never edited after creation, so it has no UpdatedAt.
type Note struct {
	NoteContent

	ID        NoteID    `json:"id"         validate:"canonical_id=note"`
	CreatedAt time.Time `json:"created_at"`
}
