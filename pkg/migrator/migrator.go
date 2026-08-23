package migrator

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Run(migrationsFS embed.FS, dir string, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection for schema setup: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS meethelper;"); err != nil {
		return fmt.Errorf("failed to create meethelper schema: %w", err)
	}

	driver, err := iofs.New(migrationsFS, dir)
	if err != nil {
		return fmt.Errorf("failed to create iofs migration driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
