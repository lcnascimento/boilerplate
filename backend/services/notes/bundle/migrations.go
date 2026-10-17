package bundle

import (
	"context"
	"database/sql"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/o11y"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
)

const migrationsTimeout = 5 * time.Minute

var migrationDirections = map[string]func(context.Context, *sql.DB) error{
	"up":   migrations.Up,
	"down": migrations.Down,
}

var ErrMigrationsUsage = errors.New("usage: migrate [direction], with direction one of the listed ones").
	WithCode("MIGRATIONS_USAGE").
	WithAttribute("directions", strings.Join(slices.Sorted(maps.Keys(migrationDirections)), ", "))

func RunMigrations(args []string) {
	if code := runMigrations(args); code != 0 {
		os.Exit(code)
	}
}

func runMigrations(args []string) int {
	ctx := context.Background()

	o11y.MustStart(ctx)

	defer func() {
		if err := o11y.Shutdown(ctx); err != nil {
			onO11yShutdownFailed(ctx, err)
		}
	}()

	signalCtx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	migrateCtx, cancel := context.WithTimeout(signalCtx, migrationsTimeout)
	defer cancel()

	if err := migrate(migrateCtx, args); err != nil {
		onMigrationsFailed(ctx, err)

		return 1
	}

	return 0
}

func migrate(ctx context.Context, args []string) error {
	direction := "up"
	if len(args) > 0 {
		direction = args[0]
	}

	run, ok := migrationDirections[direction]
	if len(args) > 1 || !ok {
		return ErrMigrationsUsage
	}

	url, err := config.DatabaseURL()
	if err != nil {
		return err
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		return ErrConnectDatabase.WithCause(err)
	}

	defer func() { _ = db.Close() }()

	return run(ctx, db)
}
