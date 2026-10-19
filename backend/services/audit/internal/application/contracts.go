package application

import (
	"context"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
)

type Validator interface {
	Struct(s any) error
}

type AuditEntriesRepository interface {
	Insert(ctx context.Context, entry *idomain.Entry) error
}
