package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend/internal/routes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

type HandlerSuite struct {
	suite.Suite

	actor    *actor.Actor
	note     *domain.Note
	noteJSON string
	service  *mocks.MockNotesService
	router   *mux.Router
}

func TestHandler(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(HandlerSuite))
}

func (s *HandlerSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.service = mocks.NewMockNotesService(ctrl)
	s.note = fixture.RandomNote()
	s.noteJSON = wireJSON(s.T(), s.note)

	orgID, err := iamdomain.ParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")
	s.Require().NoError(err)
	s.actor = actortest.Owner(orgID)

	s.router = mux.NewRouter()
	routes.RegisterNotes(s.router, s.service, withActor(s.actor))
}

func (s *HandlerSuite) TestAnswersInternalErrorWithoutAnActorInContext() {
	router := mux.NewRouter()
	routes.RegisterNotes(router, s.service, func(next http.Handler) http.Handler { return next })

	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/notes/v1/notes", `{"title":"Lista de compras"}`},
		{http.MethodGet, "/notes/v1/notes", ""},
		{http.MethodGet, "/notes/v1/notes/" + s.note.ID.String(), ""},
		{http.MethodDelete, "/notes/v1/notes/" + s.note.ID.String(), ""},
	} {
		s.Run(tc.method+" "+tc.path, func() {
			req := httptest.NewRequestWithContext(s.T().Context(), tc.method, tc.path, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			s.Require().Equal(http.StatusInternalServerError, rec.Code)
			s.Require().JSONEq(`{"code": "NO_ACTOR", "message": "no actor in context", "retryable": false}`, rec.Body.String())
		})
	}
}

func (s *HandlerSuite) TestCreateReturnsCreatedNote() {
	s.service.EXPECT().
		Create(gomock.Any(), s.actor, &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "Lista de compras", Body: "Leite, ovos e café."}}).
		Return(s.note, nil)

	rec := s.serve(http.MethodPost, "/notes/v1/notes", `{"title":"Lista de compras","body":"Leite, ovos e café."}`)

	s.Require().Equal(http.StatusCreated, rec.Code)
	s.Require().JSONEq(s.noteJSON, rec.Body.String())
}

func (s *HandlerSuite) TestCreateRejectsMalformedBody() {
	rec := s.serve(http.MethodPost, "/notes/v1/notes", `{not json`)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.Require().JSONEq(`{
		"code": "ERR_PARSE_REQUEST_BODY",
		"message": "failed to parse request body",
		"retryable": false
	}`, rec.Body.String())
}

func (s *HandlerSuite) TestCreateWritesInvalidInputWithReasons() {
	invalid := domain.ErrInvalidInput.WithCause(errors.New("[title] is a required field"))
	s.service.EXPECT().Create(gomock.Any(), s.actor, gomock.Any()).Return(nil, invalid)

	rec := s.serve(http.MethodPost, "/notes/v1/notes", `{"title":""}`)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.Require().JSONEq(`{
		"code": "INVALID_INPUT",
		"message": "invalid input",
		"retryable": false,
		"details": {"reasons": ["[title] is a required field"]}
	}`, rec.Body.String())
}

func (s *HandlerSuite) TestListReturnsEnvelope() {
	s.service.EXPECT().List(gomock.Any(), s.actor).Return([]*domain.Note{s.note}, nil)

	rec := s.serve(http.MethodGet, "/notes/v1/notes", "")

	s.Require().Equal(http.StatusOK, rec.Code)
	s.Require().JSONEq(`{"notes": [`+s.noteJSON+`]}`, rec.Body.String())
}

func (s *HandlerSuite) TestListWithoutNotesReturnsEmptyArray() {
	s.service.EXPECT().List(gomock.Any(), s.actor).Return(nil, nil)

	rec := s.serve(http.MethodGet, "/notes/v1/notes", "")

	s.Require().Equal(http.StatusOK, rec.Code)
	s.Require().JSONEq(`{"notes": []}`, rec.Body.String())
}

func (s *HandlerSuite) TestListWritesInternalErrors() {
	s.service.EXPECT().List(gomock.Any(), s.actor).Return(nil, errors.New("failed to list notes"))

	rec := s.serve(http.MethodGet, "/notes/v1/notes", "")

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
	s.Require().JSONEq(`{"code": "UNKNOWN", "message": "failed to list notes", "retryable": false}`, rec.Body.String())
}

func (s *HandlerSuite) TestGetReturnsNote() {
	s.service.EXPECT().Get(gomock.Any(), s.actor, s.note.ID).Return(s.note, nil)

	rec := s.serve(http.MethodGet, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusOK, rec.Code)
	s.Require().JSONEq(s.noteJSON, rec.Body.String())
}

func (s *HandlerSuite) TestGetUnknownNoteIsNotFound() {
	s.service.EXPECT().Get(gomock.Any(), s.actor, gomock.Any()).Return(nil, domain.ErrNoteNotFound)

	rec := s.serve(http.MethodGet, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusNotFound, rec.Code)
	s.Require().JSONEq(`{"code": "NOTE_NOT_FOUND", "message": "note not found", "retryable": false}`, rec.Body.String())
}

func (s *HandlerSuite) TestNoteOfAnotherOrganizationIsNotFound() {
	globexOwner := actortest.Owner(iamdomain.MustParseOrganizationID("org_4d7a1c3e-5b6f-4a8b-9c0d-1e2f3a4b5c6d"))
	router := mux.NewRouter()
	routes.RegisterNotes(router, s.service, withActor(globexOwner))

	s.service.EXPECT().Get(gomock.Any(), globexOwner, s.note.ID).Return(nil, notes.ErrGetNote.WithCause(domain.ErrNoteNotFound))
	s.service.EXPECT().Delete(gomock.Any(), globexOwner, s.note.ID).Return(notes.ErrDeleteNote.WithCause(domain.ErrNoteNotFound))

	for method, body := range map[string]string{
		http.MethodGet:    `{"code": "NOTE_NOT_FOUND", "message": "failed to get note", "retryable": false, "details": {"reasons": ["note not found"]}}`,
		http.MethodDelete: `{"code": "NOTE_NOT_FOUND", "message": "failed to delete note", "retryable": false, "details": {"reasons": ["note not found"]}}`,
	} {
		s.Run(method, func() {
			req := httptest.NewRequestWithContext(s.T().Context(), method, "/notes/v1/notes/"+s.note.ID.String(), nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			s.Require().Equal(http.StatusNotFound, rec.Code)
			s.Require().JSONEq(body, rec.Body.String())
		})
	}
}

func (s *HandlerSuite) TestGetRejectsMalformedID() {
	rec := s.serve(http.MethodGet, "/notes/v1/notes/not-a-note-id", "")

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.Require().JSONEq(`{
		"code": "INVALID_NOTE_ID",
		"message": "invalid note id",
		"retryable": false,
		"details": {"reasons": ["id must have prefix 'note_'"]}
	}`, rec.Body.String())
}

func (s *HandlerSuite) TestDeleteReturnsNoContent() {
	s.service.EXPECT().Delete(gomock.Any(), s.actor, s.note.ID).Return(nil)

	rec := s.serve(http.MethodDelete, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusNoContent, rec.Code)
	s.Require().Empty(rec.Body.String())
}

func (s *HandlerSuite) TestDeleteUnknownNoteIsNotFound() {
	s.service.EXPECT().Delete(gomock.Any(), s.actor, gomock.Any()).Return(domain.ErrNoteNotFound)

	rec := s.serve(http.MethodDelete, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *HandlerSuite) TestDeleteRejectsMalformedID() {
	rec := s.serve(http.MethodDelete, "/notes/v1/notes/note_bad", "")

	s.Require().Equal(http.StatusBadRequest, rec.Code)
}

func TestHandler_AnswersEachRoleOnEachRouteAsTheMatrixSays(t *testing.T) {
	t.Parallel()

	acme := iamdomain.MustParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")
	note := fixture.RandomNote()

	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	app := notes.New(storedNote{note}, v)

	const forbidden = `{"code": "ERR_UNAUTHORIZED", "message": "unauthorized request", "retryable": false}`

	endpoints := []struct{ name, method, path, body string }{
		{"create", http.MethodPost, "/notes/v1/notes", `{"title":"Lista de compras"}`},
		{"list", http.MethodGet, "/notes/v1/notes", ""},
		{"get", http.MethodGet, "/notes/v1/notes/" + note.ID.String(), ""},
		{"delete", http.MethodDelete, "/notes/v1/notes/" + note.ID.String(), ""},
	}

	matrix := map[string]struct {
		actor  *actor.Actor
		status map[string]int
	}{
		"owner":  {actortest.Owner(acme), map[string]int{"create": 201, "list": 200, "get": 200, "delete": 204}},
		"admin":  {actortest.Admin(acme), map[string]int{"create": 201, "list": 200, "get": 200, "delete": 403}},
		"viewer": {actortest.Viewer(acme), map[string]int{"create": 403, "list": 200, "get": 200, "delete": 403}},
	}

	for role, tc := range matrix {
		router := mux.NewRouter()
		routes.RegisterNotes(router, app, withActor(tc.actor))

		for _, rt := range endpoints {
			t.Run(role+" "+rt.name, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequestWithContext(t.Context(), rt.method, rt.path, strings.NewReader(rt.body))
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				require.Equal(t, tc.status[rt.name], rec.Code)

				if rec.Code == http.StatusForbidden {
					require.JSONEq(t, forbidden, rec.Body.String())
				}
			})
		}
	}
}

// storedNote is a repository holding one note, so every route that reaches it succeeds.
type storedNote struct{ note *domain.Note }

func (storedNote) Create(context.Context, *domain.Note) error { return nil }

func (r storedNote) Get(context.Context, iamdomain.OrganizationID, domain.NoteID) (*domain.Note, error) {
	return r.note, nil
}

func (r storedNote) List(context.Context, iamdomain.OrganizationID) ([]*domain.Note, error) {
	return []*domain.Note{r.note}, nil
}

func (storedNote) Delete(context.Context, iamdomain.OrganizationID, domain.NoteID) error { return nil }

func withActor(a *actor.Actor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(actortest.Context(r.Context(), a)))
		})
	}
}

func wireJSON(t *testing.T, n *domain.Note) string {
	t.Helper()

	out, err := json.Marshal(map[string]string{
		"id":         n.ID.String(),
		"title":      n.Title,
		"body":       n.Body,
		"created_at": n.CreatedAt.Format(time.RFC3339),
	})
	require.NoError(t, err)

	return string(out)
}

func (s *HandlerSuite) serve(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(s.T().Context(), method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	return rec
}
