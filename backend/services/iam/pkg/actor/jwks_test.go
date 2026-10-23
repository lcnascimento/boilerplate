package actor_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"
)

func TestMiddleware_AnswersUnauthorizedUntilTheJWKSComesUp(t *testing.T) {
	issuer := actortest.NewIssuer(t)
	jwks := newFlakyJWKS(t, issuer.Config().JWKSURL)
	cfg := issuer.Config()
	cfg.JWKSURL = jwks.server.URL

	mw := newMiddleware(t, cfg, actor.WithRefetchInterval(time.Millisecond))
	token := bearer(issuer.Mint(actortest.Admin(acmeID(t))))

	require.Equal(t, http.StatusUnauthorized, serve(t.Context(), mw, token, func(*http.Request) {}).Code)
	require.Equal(t, http.StatusUnauthorized, serve(t.Context(), mw, token, func(*http.Request) {}).Code)

	jwks.up.Store(true)

	require.Eventually(t, func() bool {
		return serve(t.Context(), mw, token, func(*http.Request) {}).Code == http.StatusOK
	}, time.Second, 10*time.Millisecond)
}

func TestMiddleware_RefetchesTheJWKSForAnUnknownKID(t *testing.T) {
	issuer := actortest.NewIssuer(t)
	mw := newMiddleware(t, issuer.Config())
	viewer := actortest.Viewer(acmeID(t))

	require.Equal(t, http.StatusOK, serve(t.Context(), mw, bearer(issuer.Mint(viewer)), func(*http.Request) {}).Code)

	issuer.Rotate()

	require.Equal(t, http.StatusOK, serve(t.Context(), mw, bearer(issuer.Mint(viewer)), func(*http.Request) {}).Code)
}

type flakyJWKS struct {
	up     atomic.Bool
	server *httptest.Server
}

func newFlakyJWKS(t *testing.T, target string) *flakyJWKS {
	t.Helper()

	f := &flakyJWKS{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !f.up.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, http.NoBody)
		if err != nil {
			t.Errorf("build JWKS request: %v", err)

			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("proxy JWKS: %v", err)

			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, resp.Body)
	}))
	t.Cleanup(f.server.Close)

	return f
}
