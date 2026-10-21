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

type Middleware struct {
	cfg  Config
	opts options

	keys        keyfunc.Keyfunc
	parser      *jwt.Parser
	stopRefresh context.CancelFunc
}

func NewMiddleware(cfg Config, opts ...Option) *Middleware {
	o := options{refetchInterval: defaultRefetchInterval}
	for _, opt := range opts {
		opt(&o)
	}

	return &Middleware{
		cfg:  cfg,
		opts: o,
		parser: jwt.NewParser(
			jwt.WithIssuer(cfg.Issuer),
			jwt.WithAudience(cfg.Audience),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
			jwt.WithLeeway(leeway),
		),
	}
}

func (m *Middleware) Start(ctx context.Context) error {
	refreshCtx, stopRefresh := context.WithCancel(context.WithoutCancel(ctx))

	keys, err := keyfunc.NewDefaultOverrideCtx(refreshCtx, []string{m.cfg.JWKSURL}, keyfunc.Override{
		HTTPTimeout:       jwksTimeout,
		RateLimitWaitMax:  jwksTimeout,
		RefreshUnknownKID: rate.NewLimiter(rate.Every(m.opts.refetchInterval), 1),
		RefreshErrorHandlerFunc: func(string) func(context.Context, error) {
			return onJWKSRefreshFailed
		},
	})
	if err != nil {
		stopRefresh()

		return ErrStartMiddleware.WithCause(err)
	}

	m.keys, m.stopRefresh = keys, stopRefresh

	onStarted(ctx)

	return nil
}

func (m *Middleware) Stop(context.Context) error {
	if m.stopRefresh != nil {
		m.stopRefresh()
	}

	return nil
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		a, failure, ok := m.authenticate(ctx, r.Header.Get("Authorization"))
		if !ok {
			onRejected(ctx, failure)
			util.WriteError(ctx, w, errors.ErrRequestUnauthenticated)

			return
		}

		ctx = onAuthenticated(newContext(ctx, a), a)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) authenticate(ctx context.Context, header string) (*Actor, reason, bool) {
	if header == "" {
		return nil, reasonMissingHeader, false
	}

	raw, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || raw == "" {
		return nil, reasonMalformed, false
	}

	var c claims
	if _, err := m.parser.ParseWithClaims(raw, &c, m.keyfunc(ctx)); err != nil {
		return nil, reasonOf(err), false
	}

	return actorOf(&c)
}

// keyfunc adds what keyfunc/v3 leaves out: it requires a kid, and accepts EdDSA only.
func (m *Middleware) keyfunc(ctx context.Context) jwt.Keyfunc {
	lookup := m.keys.KeyfuncCtx(ctx)

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

func actorOf(c *claims) (*Actor, reason, bool) {
	if c.Subject == "" || c.Org == "" || c.Role == "" || c.IssuedAt == nil {
		return nil, reasonMissingClaim, false
	}

	userID, err := domain.ParseUserID(c.Subject)
	if err != nil {
		return nil, reasonMalformed, false
	}

	orgID, err := domain.ParseOrganizationID(c.Org)
	if err != nil {
		return nil, reasonMalformed, false
	}

	role, err := roles.ParseRole(c.Role)
	if err != nil {
		return nil, reasonUnknownRole, false
	}

	return &Actor{UserID: userID, OrganizationID: orgID, Role: role}, 0, true
}
