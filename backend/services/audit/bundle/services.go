package bundle

import (
	"go.uber.org/fx"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application/audit"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport"
)

func AuditService() fx.Option {
	return fx.Module(
		"Audit Service",
		fx.Provide(fx.Annotate(audit.New, fx.As(new(transport.AuditService)))),
	)
}
