package bundle

import (
	"github.com/lcnascimento/go-kit/postgres/migrate"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/migrations"
)

func RunMigrations(args []string) {
	migrate.Run(args, migrations.FS, config.DatabaseURL)
}
