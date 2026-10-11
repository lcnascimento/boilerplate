// Package fixture builds valid, randomized domain entities for tests: a test
// overrides only the fields it is about.
package fixture

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

// maxSeed bounds the random number behind a fixture's text and age (seconds).
const maxSeed = 1_000_000

func RandomNote() *domain.Note {
	n := rand.IntN(maxSeed) //nolint:gosec // test data, not security

	return &domain.Note{
		ID:        domain.NewNoteID(),
		Title:     domain.NoteTitle(fmt.Sprintf("Note %d", n)),
		Body:      fmt.Sprintf("Body of note %d.", n),
		CreatedAt: time.Now().UTC().Add(-time.Duration(n) * time.Second).Truncate(time.Second),
	}
}
