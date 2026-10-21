package actor

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrNoActor = errors.New("no actor in context").
			WithCode("NO_ACTOR").
			WithKind(errors.KindInternal)

	ErrStartMiddleware = errors.New("failed to start the actor middleware")

	errMissingKID = errors.New("token header has no kid").WithCode("MISSING_KID")
	errBadAlg     = errors.New("token alg is not EdDSA").WithCode("BAD_ALG")
	errUnknownKID = errors.New("token kid is not in the JWKS").WithCode("UNKNOWN_KID")
)
