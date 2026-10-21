package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/http/httpserver/util"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

type Notes struct {
	service transport.NotesService
}

func New(service transport.NotesService) *Notes {
	return &Notes{service: service}
}

func (h *Notes) Create(rw http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	caller, ok := callerOf(rw, r)
	if !ok {
		return
	}

	var req models.CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(r.Context(), rw, util.ErrParseRequestBody.WithCause(err))

		return
	}

	note, err := h.service.Create(r.Context(), caller, &req)
	if err != nil {
		util.WriteError(r.Context(), rw, err)

		return
	}

	util.WriteResponse(rw, http.StatusCreated, note)
}

func (h *Notes) List(rw http.ResponseWriter, r *http.Request) {
	caller, ok := callerOf(rw, r)
	if !ok {
		return
	}

	notes, err := h.service.List(r.Context(), caller)
	if err != nil {
		util.WriteError(r.Context(), rw, err)

		return
	}

	if notes == nil {
		notes = []*domain.Note{}
	}

	util.WriteResponse(rw, http.StatusOK, &models.ListNotesResponse{Notes: notes})
}

func (h *Notes) Get(rw http.ResponseWriter, r *http.Request) {
	caller, ok := callerOf(rw, r)
	if !ok {
		return
	}

	id, err := domain.ParseNoteID(mux.Vars(r)["id"])
	if err != nil {
		util.WriteError(r.Context(), rw, err)

		return
	}

	note, err := h.service.Get(r.Context(), caller, id)
	if err != nil {
		util.WriteError(r.Context(), rw, err)

		return
	}

	util.WriteResponse(rw, http.StatusOK, note)
}

func (h *Notes) Delete(rw http.ResponseWriter, r *http.Request) {
	caller, ok := callerOf(rw, r)
	if !ok {
		return
	}

	id, err := domain.ParseNoteID(mux.Vars(r)["id"])
	if err != nil {
		util.WriteError(r.Context(), rw, err)

		return
	}

	if err := h.service.Delete(r.Context(), caller, id); err != nil {
		util.WriteError(r.Context(), rw, err)

		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

func callerOf(rw http.ResponseWriter, r *http.Request) (*actor.Actor, bool) {
	caller, err := actor.FromContext(r.Context())
	if err != nil {
		util.WriteError(r.Context(), rw, err)

		return nil, false
	}

	return caller, true
}
