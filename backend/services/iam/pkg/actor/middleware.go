package actor

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/http/httpserver/util"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

const (
	leeway = 30 * time.Second

	// jwksTimeout bounds every JWKS fetch, including the wait for the unknown-kid limiter, so a dead JWKS never holds a request long.
	jwksTimeout = 5 * time.Second

	defaultRefetchInterval = 15 * time.Second
)

type Config struct {
	JWKSURL  string
	Issuer   string
	Audience string
}

type options struct {
	refetchInterval time.Duration
}

type Option func(*options)

func WithRefetchInterval(d time.Duration) Option {
	return func(o *options) {
		o.refetchInterval = d
	}
}

type claims struct {
	jwt.RegisteredClaims

	Org  string `json:"org"`
	Role string `json:"role"`
}

type verifier struct {
	keys   keyfunc.Keyfunc
	parser *jwt.Parser
}

// NewMiddleware does not fail when the JWKS is down: it answers 401 until a fetch succeeds. ctx bounds the JWKS refresh goroutine.
func NewMiddleware(ctx context.Context, cfg Config, opts ...Option) (func(http.Handler) http.Handler, error) {
	o := options{refetchInterval: defaultRefetchInterval}
	for _, opt := range opts {
		opt(&o)
	}

	keys, err := keyfunc.NewDefaultOverrideCtx(ctx, []string{cfg.JWKSURL}, keyfunc.Override{
		HTTPTimeout:       jwksTimeout,
		RateLimitWaitMax:  jwksTimeout,
		RefreshUnknownKID: rate.NewLimiter(rate.Every(o.refetchInterval), 1),
		RefreshErrorHandlerFunc: func(string) func(context.Context, error) {
			return onJWKSRefreshFailed
		},
	})
	if err != nil {
		return nil, ErrNewMiddleware.WithCause(err)
	}

	v := &verifier{
		keys: keys,
		parser: jwt.NewParser(
			jwt.WithIssuer(cfg.Issuer),
			jwt.WithAudience(cfg.Audience),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
			jwt.WithLeeway(leeway),
		),
	}

	onMiddlewareCreated(ctx)

	return v.middleware, nil
}

func (v *verifier) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		a, failure := v.authenticate(ctx, r.Header.Get("Authorization"))
		if failure != "" {
			onRejected(ctx, failure)
			util.WriteError(ctx, w, errors.ErrRequestUnauthenticated)

			return
		}

		ctx = onAuthenticated(newContext(ctx, a), a)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (v *verifier) authenticate(ctx context.Context, header string) (*Actor, reason) {
	if header == "" {
		return nil, reasonMissingHeader
	}

	raw, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || raw == "" {
		return nil, reasonMalformed
	}

	var c claims
	if _, err := v.parser.ParseWithClaims(raw, &c, v.keyfunc(ctx)); err != nil {
		return nil, reasonOf(err)
	}

	return actorOf(&c)
}

// keyfunc adds what keyfunc/v3 leaves out: it requires a kid, and accepts EdDSA only.
func (v *verifier) keyfunc(ctx context.Context) jwt.Keyfunc {
	lookup := v.keys.KeyfuncCtx(ctx)

	return func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			return nil, errBadAlg
		}

		if kid, ok := token.Header["kid"].(string); !ok || kid == "" {
			return nil, errMissingKID
		}

		key, err := lookup(token)
		if err != nil {
			return nil, errUnknownKID.WithCause(err)
		}

		return key, nil
	}
}

func actorOf(c *claims) (*Actor, reason) {
	if c.Subject == "" || c.Org == "" || c.Role == "" || c.IssuedAt == nil {
		return nil, reasonMissingClaim
	}

	userID, err := domain.ParseUserID(c.Subject)
	if err != nil {
		return nil, reasonMalformed
	}

	orgID, err := domain.ParseOrganizationID(c.Org)
	if err != nil {
		return nil, reasonMalformed
	}

	role, err := roles.ParseRole(c.Role)
	if err != nil {
		return nil, reasonUnknownRole
	}

	return &Actor{UserID: userID, OrganizationID: orgID, Role: role}, ""
}
