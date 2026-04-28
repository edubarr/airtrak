package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("APP_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://airtrak:airtrak@localhost:5432/airtrak?sslmode=disable"
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "db/migrations"
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	err = goose.SetDialect("postgres")
	if err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	err = goose.RunContext(context.Background(), command, db, migrationsDir, os.Args[2:]...)
	if err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}

	return nil
}
