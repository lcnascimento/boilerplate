package config

import (
	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
)

var ErrDatabaseURLMissing = errors.New(
	"DATABASE_URL is not set: add the DATABASE_URL line of backend/services/notes/.env.example " +
		"to backend/services/notes/.env (`make env` creates .env only when it is missing)",
).WithCode("DATABASE_URL_MISSING")

func DatabaseURL() (string, error) {
	url := env.Get[string]("DATABASE_URL")
	if url == "" {
		return "", ErrDatabaseURLMissing
	}

	return url, nil
}
