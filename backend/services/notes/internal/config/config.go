// Package config reads the settings that both the API and cmd/migrate need,
// so each key is read in one place. A setting with no safe value fails the
// boot instead of falling back to a hidden default.
package config

import (
	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
)

// ErrDatabaseURLMissing names the exact key to add: `make env` only creates a
// missing .env, so an .env from before the key existed never gets it.
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
