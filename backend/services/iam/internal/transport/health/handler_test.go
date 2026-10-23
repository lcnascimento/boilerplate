package health_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport/health"
)

func TestHealth_Routes(t *testing.T) {
	t.Parallel()

	pass := func(context.Context) error { return nil }
	fail := func(context.Context) error { return errors.ErrMock }

	cases := map[string]struct {
		path   string
		checks []health.Check
		status int
		body   string
	}{
		"ping answers even when a check fails": {
			path:   "/ping",
			checks: []health.Check{fail},
			status: http.StatusOK,
			body:   `{"message":"pong"}`,
		},
		"health is healthy when every check passes": {
			path:   "/health",
			checks: []health.Check{pass, pass},
			status: http.StatusOK,
			body:   `{"message":"healthy"}`,
		},
		"health is unhealthy when the first check fails": {
			path:   "/health",
			checks: []health.Check{fail, pass},
			status: http.StatusServiceUnavailable,
			body:   `{"message":"unhealthy"}`,
		},
		"health is unhealthy when a later check fails": {
			path:   "/health",
			checks: []health.Check{pass, fail},
			status: http.StatusServiceUnavailable,
			body:   `{"message":"unhealthy"}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			router := mux.NewRouter()
			health.Register(router, tc.checks...)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, http.NoBody))

			require.Equal(t, tc.status, rec.Code)
			require.JSONEq(t, tc.body, rec.Body.String())
		})
	}
}
