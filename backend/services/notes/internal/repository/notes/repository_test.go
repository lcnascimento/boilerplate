package notes_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
)

type RepositorySuite struct {
	suite.Suite

	repo *notes.Repository
}

func TestRepository(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RepositorySuite))
}

func (s *RepositorySuite) SetupTest() {
	s.repo = notes.New()
}

func (s *RepositorySuite) TestStartsEmpty() {
	got, err := s.repo.List(s.T().Context())

	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().Empty(got)
}

func (s *RepositorySuite) TestCreateThenGet() {
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	s.Require().NoError(s.repo.Create(s.T().Context(), note))

	got, err := s.repo.Get(s.T().Context(), note.ID)

	s.Require().NoError(err)
	s.Require().Equal(note, got)
}

func (s *RepositorySuite) TestGetUnknownIDReturnsNotFound() {
	_, err := s.repo.Get(s.T().Context(), domain.NewNoteID())

	s.Require().ErrorIs(err, domain.ErrNoteNotFound)
}

func (s *RepositorySuite) TestStoredNoteIsNotAliasedByCaller() {
	note := newNote("original", time.Now())
	s.Require().NoError(s.repo.Create(s.T().Context(), note))

	note.Title = "mutated after create"

	got, err := s.repo.Get(s.T().Context(), note.ID)
	s.Require().NoError(err)

	got.Title = "mutated after get"

	again, err := s.repo.Get(s.T().Context(), note.ID)
	s.Require().NoError(err)
	s.Require().Equal(domain.NoteTitle("original"), again.Title)
}

func (s *RepositorySuite) TestListReturnsNewestFirst() {
	base := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	oldest := newNote("oldest", base)
	newest := newNote("newest", base.Add(2*time.Hour))
	middle := newNote("middle", base.Add(time.Hour))

	for _, note := range []*domain.Note{oldest, newest, middle} {
		s.Require().NoError(s.repo.Create(s.T().Context(), note))
	}

	got, err := s.repo.List(s.T().Context())

	s.Require().NoError(err)
	s.Require().Equal([]domain.NoteTitle{"newest", "middle", "oldest"}, titles(got))
}

func (s *RepositorySuite) TestListBreaksTimestampTiesByID() {
	at := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	first := newNote("first", at)
	second := newNote("second", at) // UUID v7: generated later, sorts higher

	s.Require().NoError(s.repo.Create(s.T().Context(), first))
	s.Require().NoError(s.repo.Create(s.T().Context(), second))

	got, err := s.repo.List(s.T().Context())

	s.Require().NoError(err)
	s.Require().Equal([]domain.NoteTitle{"second", "first"}, titles(got))
}

func (s *RepositorySuite) TestDeleteRemovesNote() {
	note := newNote("Lista de compras", time.Now())
	s.Require().NoError(s.repo.Create(s.T().Context(), note))

	s.Require().NoError(s.repo.Delete(s.T().Context(), note.ID))

	_, err := s.repo.Get(s.T().Context(), note.ID)
	s.Require().ErrorIs(err, domain.ErrNoteNotFound)
}

func (s *RepositorySuite) TestDeleteUnknownIDReturnsNotFound() {
	err := s.repo.Delete(s.T().Context(), domain.NewNoteID())

	s.Require().ErrorIs(err, domain.ErrNoteNotFound)
}

func (s *RepositorySuite) TestConcurrentAccessIsSafe() {
	const writers = 50

	var wg sync.WaitGroup

	for range writers {
		wg.Go(func() {
			note := newNote("concurrent", time.Now())
			s.NoError(s.repo.Create(s.T().Context(), note))
			_, err := s.repo.List(s.T().Context())
			s.NoError(err)
		})
	}

	wg.Wait()

	got, err := s.repo.List(s.T().Context())
	s.Require().NoError(err)
	s.Require().Len(got, writers)
}

func newNote(title domain.NoteTitle, createdAt time.Time) *domain.Note {
	note := fixture.RandomNote()
	note.Title = title
	note.CreatedAt = createdAt

	return note
}

func titles(list []*domain.Note) []domain.NoteTitle {
	out := make([]domain.NoteTitle, 0, len(list))
	for _, note := range list {
		out = append(out, note.Title)
	}

	return out
}
