package frontend_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/mocks"
)

func TestServer_HealthIsPublicAndNotesNeedAnActorToken(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	notes := mocks.NewMockNotesService(ctrl)
	db := mocks.NewMockDatabase(ctrl)
	db.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes()

	issuer := actortest.NewIssuer(t)
	authenticator := actor.NewMiddleware(issuer.Config())
	require.NoError(t, authenticator.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, authenticator.Stop(context.WithoutCancel(t.Context()))) })

	router := mux.NewRouter()
	require.NoError(t, frontend.Register(frontend.New(notes, db, authenticator), router))

	orgID, err := domain.ParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")
	require.NoError(t, err)

	owner := actortest.Owner(orgID)
	notes.EXPECT().List(gomock.Any(), owner).Return(nil, nil)

	serve := func(path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec
	}

	require.Equal(t, http.StatusOK, serve("/health", "").Code)
	require.Equal(t, http.StatusOK, serve("/ping", "").Code)

	for _, path := range []string{"/notes/v1/notes", "/notes/v1/notes/note_0199c3a1-7b2e-7c4d-9e5f-0a1b2c3d4e5f"} {
		rec := serve(path, "")
		require.Equal(t, http.StatusUnauthorized, rec.Code, path)
		require.JSONEq(t, `{"code": "ERR_UNAUTHENTICATED", "message": "unauthenticated request", "retryable": false}`, rec.Body.String(), path)
	}

	rec := serve("/notes/v1/notes", issuer.Mint(owner))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"notes": []}`, rec.Body.String())
}
