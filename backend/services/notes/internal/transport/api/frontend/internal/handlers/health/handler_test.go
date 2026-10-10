package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend/internal/routes"
)

func TestHealth_Routes(t *testing.T) {
	t.Parallel()

	router := mux.NewRouter()
	routes.RegisterHealth(router)

	cases := map[string]string{
		"/ping":   `{"message":"pong"}`,
		"/health": `{"message":"healthy"}`,
	}

	for path, body := range cases {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))

			require.Equal(t, http.StatusOK, rec.Code)
			require.JSONEq(t, body, rec.Body.String())
		})
	}
}
