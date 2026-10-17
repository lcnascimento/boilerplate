package domain

import "time"

type NoteContent struct {
	Title string `json:"title" validate:"required,max=120"`
	Body  string `json:"body"  validate:"max=10000"`
}

type Note struct {
	NoteContent

	ID        NoteID    `json:"id"         validate:"canonical_id=note"`
	CreatedAt time.Time `json:"created_at"`
}
