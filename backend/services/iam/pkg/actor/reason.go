package actor

import (
	"github.com/golang-jwt/jwt/v5"

	"github.com/lcnascimento/go-kit/errors"
)

// reason says why the middleware rejected a request. It is a metric label, so the set stays closed.
type reason string

const (
	reasonMissingHeader reason = "missing_header"
	reasonMalformed     reason = "malformed"
	reasonBadSignature  reason = "bad_signature"
	reasonUnknownKID    reason = "unknown_kid"
	reasonBadAlg        reason = "bad_alg"
	reasonBadIssuer     reason = "bad_issuer"
	reasonBadAudience   reason = "bad_audience"
	reasonExpired       reason = "expired"
	reasonMissingClaim  reason = "missing_claim"
	reasonUnknownRole   reason = "unknown_role"
)

var reasons = []reason{
	reasonMissingHeader, reasonMalformed, reasonBadSignature, reasonUnknownKID, reasonBadAlg, reasonBadIssuer,
	reasonBadAudience, reasonExpired, reasonMissingClaim, reasonUnknownRole,
}

// reasonOf maps a jwt parse error to a reason. The checks run in order, because one token can fail several of them.
func reasonOf(err error) reason {
	checks := []struct {
		target error
		reason reason
	}{
		{errMissingKID, reasonMalformed},
		{errBadAlg, reasonBadAlg},
		{errUnknownKID, reasonUnknownKID},
		{jwt.ErrTokenMalformed, reasonMalformed},
		{jwt.ErrTokenUnverifiable, reasonBadAlg},
		{jwt.ErrTokenSignatureInvalid, reasonBadSignature},
		{jwt.ErrTokenRequiredClaimMissing, reasonMissingClaim},
		{jwt.ErrTokenExpired, reasonExpired},
		{jwt.ErrTokenNotValidYet, reasonExpired},
		{jwt.ErrTokenUsedBeforeIssued, reasonExpired},
		{jwt.ErrTokenInvalidIssuer, reasonBadIssuer},
		{jwt.ErrTokenInvalidAudience, reasonBadAudience},
	}

	for _, c := range checks {
		if errors.Is(err, c.target) {
			return c.reason
		}
	}

	return reasonMalformed
}
