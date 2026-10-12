package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend/internal/routes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/mocks"
)

func TestHealth_Routes(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		path    string
		pingErr error
		pinged  bool
		status  int
		body    string
	}{
		"ping answers while the process runs": {
			path:   "/ping",
			status: http.StatusOK,
			body:   `{"message":"pong"}`,
		},
		"health is healthy when the database answers": {
			path:   "/health",
			pinged: true,
			status: http.StatusOK,
			body:   `{"message":"healthy"}`,
		},
		"health is unhealthy when the database fails": {
			path:    "/health",
			pinged:  true,
			pingErr: errors.ErrMock,
			status:  http.StatusServiceUnavailable,
			body:    `{"message":"unhealthy"}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := mocks.NewMockDatabase(gomock.NewController(t))
			if tc.pinged {
				db.EXPECT().Ping(gomock.Any()).Return(tc.pingErr)
			}

			router := mux.NewRouter()
			routes.RegisterHealth(router, db)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, http.NoBody))

			require.Equal(t, tc.status, rec.Code)
			require.JSONEq(t, tc.body, rec.Body.String())
		})
	}
}
