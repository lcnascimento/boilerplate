package actortest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
)

const (
	Audience = "boilerplate-api"
	TTL      = 5 * time.Minute

	jwksPath = "/api/auth/jwks"
)

type signingKey struct {
	kid     string
	public  ed25519.PublicKey
	private ed25519.PrivateKey
}

// Issuer stands in for the BFF: it signs Actor tokens with Ed25519 keys and serves their JWKS like Better Auth does.
type Issuer struct {
	tb     testing.TB
	server *httptest.Server

	mu   sync.Mutex
	keys []signingKey
}

func NewIssuer(tb testing.TB) *Issuer {
	tb.Helper()

	i := &Issuer{tb: tb}
	i.Rotate()

	i.server = httptest.NewServer(http.HandlerFunc(i.serveJWKS))
	tb.Cleanup(i.server.Close)

	return i
}

func (i *Issuer) Config() actor.Config {
	return actor.Config{
		JWKSURL:  i.server.URL + jwksPath,
		Issuer:   i.server.URL,
		Audience: Audience,
	}
}

// Rotate signs every later token with a new key, and keeps publishing the old ones.
func (i *Issuer) Rotate() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.keys = append(i.keys, newSigningKey(i.tb))
}

func (i *Issuer) Mint(a *actor.Actor, opts ...MintOption) string {
	i.tb.Helper()

	i.mu.Lock()
	key := i.keys[len(i.keys)-1]
	i.mu.Unlock()

	now := time.Now()
	spec := &mintSpec{
		header: map[string]any{"kid": key.kid},
		claims: jwt.MapClaims{
			"sub":  a.UserID.String(),
			"org":  a.OrganizationID.String(),
			"role": a.Role.String(),
			"iss":  i.server.URL,
			"aud":  Audience,
			"iat":  now.Unix(),
			"exp":  now.Add(TTL).Unix(),
		},
		key: key.private,
	}

	for _, opt := range opts {
		opt(i.tb, spec)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, spec.claims)
	for name, value := range spec.header {
		token.Header[name] = value
	}

	for _, name := range spec.dropHeader {
		delete(token.Header, name)
	}

	signed, err := token.SignedString(spec.key)
	require.NoError(i.tb, err)

	return signed
}

func (i *Issuer) serveJWKS(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != jwksPath {
		http.NotFound(w, r)

		return
	}

	i.mu.Lock()
	keys := make([]map[string]string, 0, len(i.keys))

	for _, k := range i.keys {
		keys = append(keys, map[string]string{
			"kid": k.kid,
			"kty": "OKP",
			"alg": "EdDSA",
			"crv": "Ed25519",
			"x":   base64.RawURLEncoding.EncodeToString(k.public),
		})
	}
	i.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{"keys": keys}); err != nil {
		i.tb.Errorf("actortest: write JWKS: %v", err)
	}
}

func newSigningKey(tb testing.TB) signingKey {
	tb.Helper()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(tb, err)

	return signingKey{kid: uuid.NewString(), public: public, private: private}
}
