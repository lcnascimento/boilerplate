package bundle

import (
	"github.com/lcnascimento/go-kit/postgres/migrate"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
)

func RunMigrations(args []string) {
	migrate.Run(args, migrations.FS, config.DatabaseURL)
}
