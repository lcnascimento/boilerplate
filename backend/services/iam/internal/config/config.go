package config

import (
	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
)

var ErrDatabaseURLMissing = errors.New(
	"DATABASE_URL is not set: run `make -C backend env` to add the missing keys of " +
		"backend/services/iam/.env.example to backend/services/iam/.env",
).WithCode("DATABASE_URL_MISSING")

func DatabaseURL() (string, error) {
	url := env.Get[string]("DATABASE_URL")
	if url == "" {
		return "", ErrDatabaseURLMissing
	}

	return url, nil
}
