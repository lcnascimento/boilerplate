package notes

import (
	"bytes"
	"context"
	"slices"
	"sync"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

// Repository keeps notes in memory; it starts empty on every boot.
// It stores and returns copies, so callers never alias stored state.
type Repository struct {
	mu    sync.RWMutex
	notes map[domain.NoteID]domain.Note
}

func New() *Repository {
	return &Repository{notes: map[domain.NoteID]domain.Note{}}
}

func (r *Repository) Create(_ context.Context, note *domain.Note) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.notes[note.ID] = *note

	return nil
}

func (r *Repository) Get(_ context.Context, id domain.NoteID) (*domain.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	note, ok := r.notes[id]
	if !ok {
		return nil, domain.ErrNoteNotFound
	}

	return &note, nil
}

// List returns every note, newest first. Equal timestamps fall back to the
// UUID v7 inside the ID, which is time-ordered.
func (r *Repository) List(_ context.Context) ([]*domain.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*domain.Note, 0, len(r.notes))
	for _, note := range r.notes {
		out = append(out, &note)
	}

	slices.SortFunc(out, func(a, b *domain.Note) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}

		bID, aID := b.ID.UUID(), a.ID.UUID()

		return bytes.Compare(bID[:], aID[:])
	})

	return out, nil
}

func (r *Repository) Delete(_ context.Context, id domain.NoteID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.notes[id]; !ok {
		return domain.ErrNoteNotFound
	}

	delete(r.notes, id)

	return nil
}
