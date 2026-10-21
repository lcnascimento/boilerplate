package actor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

func TestMiddleware_ValidTokenPutsTheActorInTheContext(t *testing.T) {
	issuer := actortest.NewIssuer(t)
	want := &actor.Actor{
		UserID:         mustUserID(t, "user_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"),
		OrganizationID: acmeID(t),
		Role:           roles.Admin,
	}

	var got *actor.Actor

	rec := serve(t.Context(), newMiddleware(t, issuer.Config()), bearer(issuer.Mint(want)), func(r *http.Request) {
		a, err := actor.FromContext(r.Context())
		require.NoError(t, err)

		got = a
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "user_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", got.UserID.String())
	require.Equal(t, "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", got.OrganizationID.String())
	require.Equal(t, roles.Admin, got.Role)
}

func TestMiddleware_RejectsWithAGeneric401AndCountsTheReason(t *testing.T) {
	cases := []struct {
		name          string
		authorization func(*actortest.Issuer, *actor.Actor) string
		reason        string
	}{
		{"no header", func(*actortest.Issuer, *actor.Actor) string { return "" }, "missing_header"},
		{"not a bearer", func(*actortest.Issuer, *actor.Actor) string { return "Basic dXNlcjpwYXNz" }, "malformed"},
		{"not a jwt", func(*actortest.Issuer, *actor.Actor) string { return "Bearer not-a-jwt" }, "malformed"},
		{"no kid", mint(actortest.WithoutHeader("kid")), "malformed"},
		{"sub without the user prefix", mint(actortest.WithClaim("sub", "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d")), "malformed"},
		{"signed by another key", mint(actortest.WithForeignKey()), "bad_signature"},
		{"kid not in the JWKS", mint(actortest.WithHeader("kid", "a6f0c2de-5b8e-4f3a-9d1c-2e7b4a6c8f10")), "unknown_kid"},
		{"alg other than EdDSA", mint(actortest.WithHeader("alg", "HS256")), "bad_alg"},
		{"another issuer", mint(actortest.WithClaim("iss", "https://evil.test")), "bad_issuer"},
		{"another audience", mint(actortest.WithClaim("aud", "other-api")), "bad_audience"},
		{"expired past the leeway", mint(actortest.WithClaim("exp", time.Now().Add(-time.Minute).Unix())), "expired"},
		{"no org", mint(actortest.WithoutClaim("org")), "missing_claim"},
		{"no iat", mint(actortest.WithoutClaim("iat")), "missing_claim"},
		{"unknown role", mint(actortest.WithClaim("role", "member")), "unknown_role"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issuer := actortest.NewIssuer(t)
			authorization := tc.authorization(issuer, actortest.Owner(acmeID(t)))
			mw := newMiddleware(t, issuer.Config())
			before := failures(t, tc.reason)

			var rec *httptest.ResponseRecorder

			span := inSpan(t, func(ctx context.Context) {
				rec = serve(ctx, mw, authorization, func(*http.Request) {
					t.Error("the middleware let the request through")
				})
			})

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			require.JSONEq(t, unauthenticatedBody, rec.Body.String())
			require.Equal(t, before+1, failures(t, tc.reason))
			require.Equal(t, tc.reason, stringAttr(span, "iam.authentication.failure_reason"))
		})
	}
}

func TestMiddleware_AcceptsClockSkewWithinTheLeeway(t *testing.T) {
	issuer := actortest.NewIssuer(t)
	token := issuer.Mint(
		actortest.Viewer(acmeID(t)),
		actortest.WithClaim("exp", time.Now().Add(-10*time.Second).Unix()),
		actortest.WithClaim("iat", time.Now().Add(10*time.Second).Unix()),
	)

	rec := serve(t.Context(), newMiddleware(t, issuer.Config()), bearer(token), func(*http.Request) {})

	require.Equal(t, http.StatusOK, rec.Code)
}

const unauthenticatedBody = `{"code":"ERR_UNAUTHENTICATED","message":"unauthenticated request","retryable":false}`

func mint(opts ...actortest.MintOption) func(*actortest.Issuer, *actor.Actor) string {
	return func(issuer *actortest.Issuer, a *actor.Actor) string {
		return bearer(issuer.Mint(a, opts...))
	}
}

func newMiddleware(t *testing.T, cfg actor.Config, opts ...actor.Option) func(http.Handler) http.Handler {
	t.Helper()

	mw, err := actor.NewMiddleware(t.Context(), cfg, opts...)
	require.NoError(t, err)

	return mw
}

func serve(ctx context.Context, mw func(http.Handler) http.Handler, authorization string, next func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/notes/v1/notes", http.NoBody)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(r)
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	return rec
}

func bearer(token string) string {
	return "Bearer " + token
}

func mustUserID(t *testing.T, s string) domain.UserID {
	t.Helper()

	id, err := domain.ParseUserID(s)
	require.NoError(t, err)

	return id
}

func acmeID(t *testing.T) domain.OrganizationID {
	t.Helper()

	id, err := domain.ParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")
	require.NoError(t, err)

	return id
}
