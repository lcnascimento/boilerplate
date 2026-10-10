package domain

import "time"

// Note is never edited after creation, so it has no UpdatedAt.
type Note struct {
	ID        NoteID    `json:"id"`
	Title     NoteTitle `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (n *Note) Validate() error {
	if err := n.ID.Validate(); err != nil {
		return err
	}

	return n.Title.Validate()
}
