package transport

import (
	"context"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
)

type AuditService interface {
	Record(ctx context.Context, entry *idomain.Entry) error
}

type Database interface {
	Ping(ctx context.Context) error
}

type Consumer interface {
	Ready(ctx context.Context) error
}
