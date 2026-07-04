package store

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func RunMigrations(dsn string) error {
	return runMigrations(dsn, "")
}

func RunMigrationsInSchema(dsn, schemaName string) error {
	return runMigrations(dsn, schemaName)
}

func runMigrations(dsn, schemaName string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("cannot connect to database: %w", err)
	}
	defer db.Close()

	cfg := &postgres.Config{}
	if schemaName != "" {
		cfg.SchemaName = schemaName
	}

	driver, err := postgres.WithInstance(db, cfg)
	if err != nil {
		return fmt.Errorf("cannot create postgres driver: %w", err)
	}

	source, err := iofs.New(MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("cannot create iofs source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("cannot create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("cannot run migrations: %w", err)
	}

	return nil
}
