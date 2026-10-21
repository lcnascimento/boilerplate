package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/consumer"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/health"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/internal/mocks"
)

func TestHealth_Routes(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		path     string
		pingErr  error
		readyErr error
		status   int
		body     string
	}{
		"ping answers while the process runs": {
			path:   "/ping",
			status: http.StatusOK,
			body:   `{"message":"pong"}`,
		},
		"health is healthy when the database answers and the consumer is ready": {
			path:   "/health",
			status: http.StatusOK,
			body:   `{"message":"healthy"}`,
		},
		"health is unhealthy when the database fails": {
			path:    "/health",
			pingErr: errors.ErrMock,
			status:  http.StatusServiceUnavailable,
			body:    `{"message":"unhealthy"}`,
		},
		"health is unhealthy when the consumer is not consuming": {
			path:     "/health",
			readyErr: consumer.ErrNotConsuming,
			status:   http.StatusServiceUnavailable,
			body:     `{"message":"unhealthy"}`,
		},
		"health is unhealthy when the consumer cannot reach Kafka": {
			path:     "/health",
			readyErr: events.ErrKafkaUnavailable,
			status:   http.StatusServiceUnavailable,
			body:     `{"message":"unhealthy"}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mocks.NewMockDatabase(ctrl)
			db.EXPECT().Ping(gomock.Any()).Return(tc.pingErr).AnyTimes()

			ready := mocks.NewMockConsumer(ctrl)
			ready.EXPECT().Ready(gomock.Any()).Return(tc.readyErr).AnyTimes()

			router := mux.NewRouter()
			health.Register(router, db, ready)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, http.NoBody))

			require.Equal(t, tc.status, rec.Code)
			require.JSONEq(t, tc.body, rec.Body.String())
		})
	}
}
