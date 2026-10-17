package config

import (
	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
)

var (
	ErrDatabaseURLMissing       = errMissing("DATABASE_URL").WithCode("DATABASE_URL_MISSING")
	ErrSchemaRegistryURLMissing = errMissing("SCHEMA_REGISTRY_URL").WithCode("SCHEMA_REGISTRY_URL_MISSING")
)

func errMissing(key string) errors.CustomError {
	return errors.New(
		"%s is not set: add the %s line of backend/services/notes/.env.example "+
			"to backend/services/notes/.env (`make env` creates .env only when it is missing)",
		key, key,
	)
}

func DatabaseURL() (string, error) {
	return required("DATABASE_URL", ErrDatabaseURLMissing)
}

func SchemaRegistryURL() (string, error) {
	return required("SCHEMA_REGISTRY_URL", ErrSchemaRegistryURLMissing)
}

func required(key string, missing error) (string, error) {
	value := env.Get[string](key)
	if value == "" {
		return "", missing
	}

	return value, nil
}
