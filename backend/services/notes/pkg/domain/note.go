package domain

import (
	"time"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
)

type NoteContent struct {
	Title string `json:"title" validate:"required,max=120"`
	Body  string `json:"body"  validate:"max=10000"`
}

type Note struct {
	NoteContent

	ID             NoteID                   `json:"id"         validate:"canonical_id=note"`
	OrganizationID iamdomain.OrganizationID `json:"-"`
	CreatedAt      time.Time                `json:"created_at"`
}
