package actor

import (
	"github.com/golang-jwt/jwt/v5"

	"github.com/lcnascimento/go-kit/errors"
)

type reason int

const (
	reasonMissingHeader reason = iota
	reasonMalformed
	reasonBadSignature
	reasonUnknownKID
	reasonBadAlg
	reasonBadIssuer
	reasonBadAudience
	reasonExpired
	reasonMissingClaim
	reasonUnknownRole

	reasonCount
)

var reasonLabels = [reasonCount]string{
	reasonMissingHeader: "missing_header",
	reasonMalformed:     "malformed",
	reasonBadSignature:  "bad_signature",
	reasonUnknownKID:    "unknown_kid",
	reasonBadAlg:        "bad_alg",
	reasonBadIssuer:     "bad_issuer",
	reasonBadAudience:   "bad_audience",
	reasonExpired:       "expired",
	reasonMissingClaim:  "missing_claim",
	reasonUnknownRole:   "unknown_role",
}

func (r reason) String() string {
	return reasonLabels[r]
}

var reasonsByCauseInPrecedence = []struct {
	cause  error
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

func reasonOf(err error) reason {
	for _, c := range reasonsByCauseInPrecedence {
		if errors.Is(err, c.cause) {
			return c.reason
		}
	}

	return reasonMalformed
}
