package health_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/health"
)

func TestServer_HealthRunsEveryCheck(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		second func(context.Context) error
		status int
		body   string
	}{
		"healthy when every check passes": {
			second: pass,
			status: http.StatusOK,
			body:   `{"message":"healthy"}`,
		},
		"unhealthy when a later check fails": {
			second: fail,
			status: http.StatusServiceUnavailable,
			body:   `{"message":"unhealthy"}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			port := freePort(t)
			server := health.NewServer(port, func() { t.Error("health server stopped unexpectedly") }, pass, tc.second)
			require.NoError(t, server.Start(t.Context()))
			t.Cleanup(func() { _ = server.Shutdown(context.WithoutCancel(t.Context())) })

			status, body := get(t, "http://localhost:"+strconv.Itoa(port)+"/health")

			require.Equal(t, tc.status, status)
			require.JSONEq(t, tc.body, body)
		})
	}
}

func pass(context.Context) error { return nil }

func fail(context.Context) error { return errors.ErrMock }

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "localhost:0")
	require.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	return port
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()

	var (
		status int
		body   string
	)

	require.Eventually(t, func() bool {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
		if err != nil {
			return false
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return false
		}

		defer func() { _ = res.Body.Close() }()

		raw, err := io.ReadAll(res.Body)
		if err != nil {
			return false
		}

		status, body = res.StatusCode, string(raw)

		return true
	}, 5*time.Second, 20*time.Millisecond)

	return status, body
}
