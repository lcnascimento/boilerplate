package bundle

import (
	"go.uber.org/fx"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/outbox"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
)

func NotesService() fx.Option {
	return fx.Module(
		"Notes Service",
		fx.Provide(fx.Annotate(notes.New, fx.As(new(transport.NotesService)))),
	)
}

func OutboxService() fx.Option {
	return fx.Module(
		"Outbox Service",
		fx.Provide(fx.Annotate(outbox.New, fx.As(new(transport.OutboxService)))),
	)
}
