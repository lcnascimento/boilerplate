package notes_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/mock/gomock"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

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

var (
	now     = time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	metrics = sdkmetric.NewManualReader()
)

func TestMain(m *testing.M) {
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics)))

	os.Exit(m.Run())
}

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
	repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

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

var acme = iamdomain.MustParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")

func owner(t *testing.T) *actor.Actor {
	t.Helper()

	return actortest.Owner(acme)
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

	s.repo.EXPECT().Create(gomock.Any(), s.owner, gomock.Any()).DoAndReturn(func(_ context.Context, _ *actor.Actor, note *domain.Note) error {
		stored = note

		return nil
	})

	got, err := s.app.Create(s.T().Context(), s.owner, &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "Lista de compras", Body: "Leite."}})

	s.Require().NoError(err)
	s.Require().True(strings.HasPrefix(got.ID.String(), "note_"), got.ID.String())
	s.Require().Equal("Lista de compras", got.Title)
	s.Require().Equal("Leite.", got.Body)
	s.Require().Equal(now, got.CreatedAt)
	s.Require().Equal("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", got.OrganizationID.String())
	s.Require().Equal(got, stored)
}

func (s *ApplicationSuite) TestCreateWithoutBodyKeepsEmptyBody() {
	s.repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

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
	s.repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.ErrMock)

	_, err := s.app.Create(s.T().Context(), s.owner, &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "a"}})

	s.Require().ErrorIs(err, notes.ErrCreateNote)
	s.Require().ErrorIs(err, errors.ErrMock)
}

func (s *ApplicationSuite) TestListReturnsRepositoryOrder() {
	newest, oldest := fixture.RandomNote(), fixture.RandomNote()
	s.repo.EXPECT().List(gomock.Any(), acme).Return([]*domain.Note{newest, oldest}, nil)

	got, err := s.app.List(s.T().Context(), s.owner)

	s.Require().NoError(err)
	s.Require().Equal([]*domain.Note{newest, oldest}, got)
}

func (s *ApplicationSuite) TestListWrapsRepositoryErrors() {
	s.repo.EXPECT().List(gomock.Any(), acme).Return(nil, errors.ErrMock)

	got, err := s.app.List(s.T().Context(), s.owner)

	s.Require().ErrorIs(err, notes.ErrListNotes)
	s.Require().ErrorIs(err, errors.ErrMock)
	s.Require().NotNil(got)
	s.Require().Empty(got)
}

func (s *ApplicationSuite) TestGetReturnsNote() {
	note := fixture.RandomNote()
	s.repo.EXPECT().Get(gomock.Any(), acme, note.ID).Return(note, nil)

	got, err := s.app.Get(s.T().Context(), s.owner, note.ID)

	s.Require().NoError(err)
	s.Require().Equal(note, got)
}

func (s *ApplicationSuite) TestGetUnknownNoteIsNotFound() {
	id := domain.NewNoteID()
	s.repo.EXPECT().Get(gomock.Any(), acme, id).Return(nil, domain.ErrNoteNotFound)

	_, err := s.app.Get(s.T().Context(), s.owner, id)

	s.Require().ErrorIs(err, notes.ErrGetNote)
	s.Require().ErrorIs(err, domain.ErrNoteNotFound)
	s.Require().Equal(errors.KindNotFound, errors.Kind(err))
}

func (s *ApplicationSuite) TestDeleteRemovesNote() {
	id := domain.NewNoteID()
	s.repo.EXPECT().Delete(gomock.Any(), s.owner, id).Return(nil)

	s.Require().NoError(s.app.Delete(s.T().Context(), s.owner, id))
}

func (s *ApplicationSuite) TestDeleteUnknownNoteIsNotFound() {
	id := domain.NewNoteID()
	s.repo.EXPECT().Delete(gomock.Any(), s.owner, id).Return(domain.ErrNoteNotFound)

	err := s.app.Delete(s.T().Context(), s.owner, id)

	s.Require().ErrorIs(err, notes.ErrDeleteNote)
	s.Require().ErrorIs(err, domain.ErrNoteNotFound)
}

func (s *ApplicationSuite) TestDeleteOrganizationNotesDeletesThroughTheRepository() {
	s.repo.EXPECT().DeleteByOrganization(gomock.Any(), acme).Return(int64(3), nil)

	s.Require().NoError(s.app.DeleteOrganizationNotes(s.T().Context(), acme))
}

func (s *ApplicationSuite) TestDeleteOrganizationNotesWrapsRepositoryErrors() {
	s.repo.EXPECT().DeleteByOrganization(gomock.Any(), acme).Return(int64(0), errors.ErrMock)

	err := s.app.DeleteOrganizationNotes(s.T().Context(), acme)

	s.Require().ErrorIs(err, notes.ErrDeleteOrganizationNotes)
	s.Require().ErrorIs(err, errors.ErrMock)
}

func (s *ApplicationSuite) TestDeniesWhatTheRoleLacksBeforeTheRepositoryEvenForAMissingNote() {
	viewer, admin := actortest.Viewer(acme), actortest.Admin(acme)
	missing := domain.NewNoteID()
	create := &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "Lista de compras"}}

	cases := map[string]struct {
		call       func() error
		operation  error
		permission string
		role       string
	}{
		"viewer creates": {
			call: func() error {
				_, err := s.app.Create(s.T().Context(), viewer, create)

				return err
			},
			operation:  notes.ErrCreateNote,
			permission: "note:create",
			role:       "viewer",
		},
		"viewer deletes": {
			call:       func() error { return s.app.Delete(s.T().Context(), viewer, missing) },
			operation:  notes.ErrDeleteNote,
			permission: "note:delete",
			role:       "viewer",
		},
		"admin deletes": {
			call:       func() error { return s.app.Delete(s.T().Context(), admin, missing) },
			operation:  notes.ErrDeleteNote,
			permission: "note:delete",
			role:       "admin",
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			before := denials(s.T(), tc.permission, tc.role)

			err := tc.call()

			s.Require().ErrorIs(err, tc.operation)
			s.Require().ErrorIs(err, errors.ErrRequestUnauthorized)
			s.Require().Equal(errors.KindUnauthorized, errors.Kind(err))
			s.Require().Equal(before+1, denials(s.T(), tc.permission, tc.role))
		})
	}
}

func (s *ApplicationSuite) TestEveryRoleReadsNotes() {
	note := fixture.RandomNote()

	for _, a := range []*actor.Actor{actortest.Owner(acme), actortest.Admin(acme), actortest.Viewer(acme)} {
		s.Run(a.Role.String(), func() {
			s.repo.EXPECT().List(gomock.Any(), acme).Return([]*domain.Note{note}, nil)
			s.repo.EXPECT().Get(gomock.Any(), acme, note.ID).Return(note, nil)

			listed, err := s.app.List(s.T().Context(), a)
			s.Require().NoError(err)
			s.Require().Equal([]*domain.Note{note}, listed)

			got, err := s.app.Get(s.T().Context(), a, note.ID)
			s.Require().NoError(err)
			s.Require().Equal(note, got)
		})
	}
}

func (s *ApplicationSuite) TestAdminCreatesNotes() {
	s.repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	got, err := s.app.Create(s.T().Context(), actortest.Admin(acme), &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "Ideias"}})

	s.Require().NoError(err)
	s.Require().Equal("Ideias", got.Title)
}

func denials(t *testing.T, permission, role string) int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "iam.actor.authorization.denials" {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "iam.actor.authorization.denials is a %T", m.Data)

			for _, dp := range sum.DataPoints {
				p, _ := dp.Attributes.Value("permission")
				r, _ := dp.Attributes.Value("role")

				if p.AsString() == permission && r.AsString() == role {
					return dp.Value
				}
			}
		}
	}

	return 0
}
