package fixture

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

const maxSeed = 1_000_000

func RandomNote() *domain.Note {
	n := rand.IntN(maxSeed) //nolint:gosec // test data, not security

	return &domain.Note{
		ID:             domain.NewNoteID(),
		OrganizationID: iamdomain.OrganizationIDFromUUID(uuid.New()),
		NoteContent: domain.NoteContent{
			Title: fmt.Sprintf("Note %d", n),
			Body:  fmt.Sprintf("Body of note %d.", n),
		},
		CreatedAt: time.Now().UTC().Add(-time.Duration(n) * time.Second).Truncate(time.Second),
	}
}
