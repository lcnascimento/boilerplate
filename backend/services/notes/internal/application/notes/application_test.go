package notes_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

var now = time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)

type ApplicationSuite struct {
	suite.Suite

	repo  *mocks.MockNotesRepository
	app   *notes.Application
	owner *actor.Actor
}

func TestApplication(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ApplicationSuite))
}

func TestApplication_DefaultClockIsUTCInMicroseconds(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := mocks.NewMockNotesRepository(ctrl)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	before := time.Now()
	got, err := notes.New(repo, v).Create(t.Context(), owner(t), &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "a"}})
	after := time.Now()

	require.NoError(t, err)
	require.Equal(t, time.UTC, got.CreatedAt.Location())
	require.Zero(t, got.CreatedAt.Nanosecond()%int(time.Microsecond))
	require.WithinRange(t, got.CreatedAt, before.Truncate(time.Microsecond), after)
}

func owner(t *testing.T) *actor.Actor {
	t.Helper()

	orgID, err := iamdomain.ParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")
	require.NoError(t, err)

	return actortest.Owner(orgID)
}

func (s *ApplicationSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.repo = mocks.NewMockNotesRepository(ctrl)

	v, err := validator.New(validator.WithJSONFieldNames())
	s.Require().NoError(err)

	s.app = notes.New(s.repo, v, notes.WithClock(func() time.Time { return now }))

	s.owner = owner(s.T())
}

func (s *ApplicationSuite) TestCreateStoresAndReturnsNote() {
	var stored *domain.Note

	s.repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, note *domain.Note) error {
		stored = note

		return nil
	})

	got, err := s.app.Create(s.T().Context(), s.owner, &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "Lista de compras", Body: "Leite."}})

	s.Require().NoError(err)
	s.Require().True(strings.HasPrefix(got.ID.String(), "note_"), got.ID.String())
	s.Require().Equal("Lista de compras", got.Title)
	s.Require().Equal("Leite.", got.Body)
	s.Require().Equal(now, got.CreatedAt)
	s.Require().Equal(got, stored)
}

func (s *ApplicationSuite) TestCreateWithoutBodyKeepsEmptyBody() {
	s.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	got, err := s.app.Create(s.T().Context(), s.owner, &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "Ideias"}})

	s.Require().NoError(err)
	s.Require().Empty(got.Body)
}

func (s *ApplicationSuite) TestCreateRejectsInvalidInputWithEveryReason() {
	cases := map[string]struct {
		req     *models.CreateNoteRequest
		reasons []string
	}{
		"nil request": {
			req:     nil,
			reasons: []string{"request body is required"},
		},
		"empty title": {
			req:     &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: ""}},
			reasons: []string{"[title] is a required field"},
		},
		"title over 120 characters": {
			req:     &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: strings.Repeat("a", 121)}},
			reasons: []string{"[title] must be a maximum of 120 characters in length"},
		},
		"body over 10000 characters": {
			req:     &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "a", Body: strings.Repeat("a", 10_001)}},
			reasons: []string{"[body] must be a maximum of 10,000 characters in length"},
		},
		"empty title and long body": {
			req: &models.CreateNoteRequest{NoteContent: domain.NoteContent{Body: strings.Repeat("a", 10_001)}},
			reasons: []string{
				"[title] is a required field",
				"[body] must be a maximum of 10,000 characters in length",
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			_, err := s.app.Create(s.T().Context(), s.owner, tc.req)

			s.Require().ErrorIs(err, domain.ErrInvalidInput)
			s.Require().Equal(errors.KindInvalidInput, errors.Kind(err))
			s.Require().Equal(errors.CodeType("INVALID_INPUT"), errors.Code(err))
			s.Require().Equal(tc.reasons, errors.SafeReasons(err))
		})
	}
}

// go-kit errors before v1.9.1 wrote validator failures into a shared map.
func (s *ApplicationSuite) TestCreateValidatesConcurrently() {
	var wg sync.WaitGroup

	for range 20 {
		wg.Go(func() {
			_, err := s.app.Create(s.T().Context(), s.owner, &models.CreateNoteRequest{})
			s.ErrorIs(err, domain.ErrInvalidInput)
		})
	}

	wg.Wait()
}

func (s *ApplicationSuite) TestCreateWrapsRepositoryErrors() {
	s.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.ErrMock)

	_, err := s.app.Create(s.T().Context(), s.owner, &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "a"}})

	s.Require().ErrorIs(err, notes.ErrCreateNote)
	s.Require().ErrorIs(err, errors.ErrMock)
}

func (s *ApplicationSuite) TestListReturnsRepositoryOrder() {
	newest, oldest := fixture.RandomNote(), fixture.RandomNote()
	s.repo.EXPECT().List(gomock.Any()).Return([]*domain.Note{newest, oldest}, nil)

	got, err := s.app.List(s.T().Context(), s.owner)

	s.Require().NoError(err)
	s.Require().Equal([]*domain.Note{newest, oldest}, got)
}

func (s *ApplicationSuite) TestListWrapsRepositoryErrors() {
	s.repo.EXPECT().List(gomock.Any()).Return(nil, errors.ErrMock)

	got, err := s.app.List(s.T().Context(), s.owner)

	s.Require().ErrorIs(err, notes.ErrListNotes)
	s.Require().ErrorIs(err, errors.ErrMock)
	s.Require().NotNil(got)
	s.Require().Empty(got)
}

func (s *ApplicationSuite) TestGetReturnsNote() {
	note := fixture.RandomNote()
	s.repo.EXPECT().Get(gomock.Any(), note.ID).Return(note, nil)

	got, err := s.app.Get(s.T().Context(), s.owner, note.ID)

	s.Require().NoError(err)
	s.Require().Equal(note, got)
}

func (s *ApplicationSuite) TestGetUnknownNoteIsNotFound() {
	id := domain.NewNoteID()
	s.repo.EXPECT().Get(gomock.Any(), id).Return(nil, domain.ErrNoteNotFound)

	_, err := s.app.Get(s.T().Context(), s.owner, id)

	s.Require().ErrorIs(err, notes.ErrGetNote)
	s.Require().ErrorIs(err, domain.ErrNoteNotFound)
	s.Require().Equal(errors.KindNotFound, errors.Kind(err))
}

func (s *ApplicationSuite) TestDeleteRemovesNote() {
	id := domain.NewNoteID()
	s.repo.EXPECT().Delete(gomock.Any(), id).Return(nil)

	s.Require().NoError(s.app.Delete(s.T().Context(), s.owner, id))
}

func (s *ApplicationSuite) TestDeleteUnknownNoteIsNotFound() {
	id := domain.NewNoteID()
	s.repo.EXPECT().Delete(gomock.Any(), id).Return(domain.ErrNoteNotFound)

	err := s.app.Delete(s.T().Context(), s.owner, id)

	s.Require().ErrorIs(err, notes.ErrDeleteNote)
	s.Require().ErrorIs(err, domain.ErrNoteNotFound)
}
