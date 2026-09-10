package migration

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// newMigrator builds a migrator over the embedded migrations. Callers must not
// Close it: that would close the shared *sql.DB through the SQLite driver, so
// the application remains responsible for closing the database.
func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create embedded migration source: %w", err)
	}
	databaseDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("create sqlite migration driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", databaseDriver)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return migrator, nil
}

// Migrate applies every embedded schema migration that has not yet run.
func Migrate(ctx context.Context, db *sql.DB) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("check migration context: %w", err)
	}

	migration, err := newMigrator(db)
	if err != nil {
		return err
	}

	err = migration.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("no db migrations to apply")
		return nil
	}
	if err != nil {
		return fmt.Errorf("apply db migrations: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("check migration context: %w", err)
	}

	slog.Info("db migrations applied")

	return nil
}
