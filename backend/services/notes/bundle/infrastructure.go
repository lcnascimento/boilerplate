package bundle

import (
	"log/slog"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
)

const pkg = "notes/bundle"

func Logger(logger *slog.Logger) fx.Option {
	return fx.Module(
		"Logger",
		fx.Provide(func() *slog.Logger { return logger }),
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
	)
}

func Validator() fx.Option {
	return fx.Module(
		"Validator",
		fx.Provide(fx.Annotate(
			func() (*validator.Validator, error) {
				return validator.New(validator.WithJSONFieldNames())
			},
			fx.As(new(application.Validator)),
		)),
	)
}

func Repositories() fx.Option {
	return fx.Module(
		"Repositories",
		fx.Provide(fx.Annotate(notes.New, fx.As(new(application.NotesRepository)))),
	)
}
