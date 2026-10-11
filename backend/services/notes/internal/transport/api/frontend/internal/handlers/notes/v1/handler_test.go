package v1_test

import (
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

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend/internal/routes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

type HandlerSuite struct {
	suite.Suite

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

	s.router = mux.NewRouter()
	routes.RegisterNotes(s.router, s.service)
}

func (s *HandlerSuite) TestCreateReturnsCreatedNote() {
	s.service.EXPECT().
		Create(gomock.Any(), &models.CreateNoteRequest{NoteContent: domain.NoteContent{Title: "Lista de compras", Body: "Leite, ovos e café."}}).
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
	s.service.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, invalid)

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
	s.service.EXPECT().List(gomock.Any()).Return([]*domain.Note{s.note}, nil)

	rec := s.serve(http.MethodGet, "/notes/v1/notes", "")

	s.Require().Equal(http.StatusOK, rec.Code)
	s.Require().JSONEq(`{"notes": [`+s.noteJSON+`]}`, rec.Body.String())
}

func (s *HandlerSuite) TestListWithoutNotesReturnsEmptyArray() {
	s.service.EXPECT().List(gomock.Any()).Return(nil, nil)

	rec := s.serve(http.MethodGet, "/notes/v1/notes", "")

	s.Require().Equal(http.StatusOK, rec.Code)
	s.Require().JSONEq(`{"notes": []}`, rec.Body.String())
}

func (s *HandlerSuite) TestListWritesInternalErrors() {
	s.service.EXPECT().List(gomock.Any()).Return(nil, errors.New("failed to list notes"))

	rec := s.serve(http.MethodGet, "/notes/v1/notes", "")

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
	s.Require().JSONEq(`{"code": "UNKNOWN", "message": "failed to list notes", "retryable": false}`, rec.Body.String())
}

func (s *HandlerSuite) TestGetReturnsNote() {
	s.service.EXPECT().Get(gomock.Any(), s.note.ID).Return(s.note, nil)

	rec := s.serve(http.MethodGet, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusOK, rec.Code)
	s.Require().JSONEq(s.noteJSON, rec.Body.String())
}

func (s *HandlerSuite) TestGetUnknownNoteIsNotFound() {
	s.service.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNoteNotFound)

	rec := s.serve(http.MethodGet, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusNotFound, rec.Code)
	s.Require().JSONEq(`{"code": "NOTE_NOT_FOUND", "message": "note not found", "retryable": false}`, rec.Body.String())
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
	s.service.EXPECT().Delete(gomock.Any(), s.note.ID).Return(nil)

	rec := s.serve(http.MethodDelete, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusNoContent, rec.Code)
	s.Require().Empty(rec.Body.String())
}

func (s *HandlerSuite) TestDeleteUnknownNoteIsNotFound() {
	s.service.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(domain.ErrNoteNotFound)

	rec := s.serve(http.MethodDelete, "/notes/v1/notes/"+s.note.ID.String(), "")

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *HandlerSuite) TestDeleteRejectsMalformedID() {
	rec := s.serve(http.MethodDelete, "/notes/v1/notes/note_bad", "")

	s.Require().Equal(http.StatusBadRequest, rec.Code)
}

// wireJSON spells the wire contract of a note field by field, independent of
// the domain's JSON encoding.
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
