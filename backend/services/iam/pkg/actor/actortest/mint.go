package actortest

import (
	"crypto/ed25519"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

type mintSpec struct {
	header     map[string]any
	dropHeader []string
	claims     jwt.MapClaims
	key        ed25519.PrivateKey
}

type MintOption func(testing.TB, *mintSpec)

func WithClaim(name string, value any) MintOption {
	return func(_ testing.TB, s *mintSpec) {
		s.claims[name] = value
	}
}

func WithoutClaim(name string) MintOption {
	return func(_ testing.TB, s *mintSpec) {
		delete(s.claims, name)
	}
}

func WithHeader(name string, value any) MintOption {
	return func(_ testing.TB, s *mintSpec) {
		s.header[name] = value
	}
}

func WithoutHeader(name string) MintOption {
	return func(_ testing.TB, s *mintSpec) {
		s.dropHeader = append(s.dropHeader, name)
	}
}

// WithForeignKey signs with a key the JWKS does not publish, under the kid of the current one.
func WithForeignKey() MintOption {
	return func(tb testing.TB, s *mintSpec) {
		tb.Helper()

		s.key = newSigningKey(tb).private
	}
}
