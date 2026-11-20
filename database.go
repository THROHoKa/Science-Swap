package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

func openDatabase(url string) (*sql.DB, error) {
	config, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, errors.New("invalid PostgreSQL connection string")
	}
	// Keep queries compatible with PgBouncer transaction pooling.
	config.DefaultQueryExecMode = pgx.QueryExecModeExec
	return stdlib.OpenDB(*config), nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	migrations, err := fs.Sub(files, "migrations")
	if err != nil {
		return err
	}
	locker, err := lock.NewPostgresTableLocker()
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, goose.WithLocker(locker))
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	log.Printf("Applied %d migration(s)", len(results))
	return nil
}
