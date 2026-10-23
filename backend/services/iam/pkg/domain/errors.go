package domain

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrInvalidUserID = errors.New("invalid user id").
				WithCode("INVALID_USER_ID").
				WithKind(errors.KindInvalidInput)

	ErrInvalidOrganizationID = errors.New("invalid organization id").
					WithCode("INVALID_ORGANIZATION_ID").
					WithKind(errors.KindInvalidInput)
)
