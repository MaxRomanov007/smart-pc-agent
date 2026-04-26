package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"smart-pc-agent/internal/config"
	appStorage "smart-pc-agent/internal/storage/sqlite/app-storage"
	commandParameters "smart-pc-agent/internal/storage/sqlite/command-parameters"
	"smart-pc-agent/internal/storage/sqlite/commands"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	AppStorage        *appStorage.Storage
	Commands          *commands.Storage
	CommandParameters *commandParameters.Storage
	queries           *dbqueries.Queries
}

func New(ctx context.Context, log *slog.Logger, cfg config.Storage) (*Storage, error) {
	const op = "storage.sqlite.New"

	if err := preventDatabaseFileCreated(cfg.Path); err != nil {
		return nil, fmt.Errorf("%s: failed to prevent database file created: %w", op, err)
	}

	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to connect to database: %w", op, err)
	}
	go func() {
		<-ctx.Done()
		if err := db.Close(); err != nil {
			log.Error("failed to close sqlite database", sl.Err(err))
		}
	}()

	if err := migrate(db, ctx); err != nil {
		return nil, fmt.Errorf("%s: failed to apply migrations: %w", op, err)
	}

	queries := dbqueries.New(db)

	return &Storage{
		AppStorage:        appStorage.New(queries),
		Commands:          commands.New(db),
		CommandParameters: commandParameters.New(queries),
		queries:           queries,
	}, nil
}

func preventDatabaseFileCreated(path string) error {
	const op = "storage.sqlite.preventDatabaseFileCreated"

	if filepath.Ext(path) != ".db" {
		return fmt.Errorf("%s: %q not a database file", op, path)
	}

	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return fmt.Errorf("%s: failed to create directories to path: %w", op, err)
		}

		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("%s: failed to create database file: %w", op, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("%s: failed to close database file: %w", op, err)
		}

		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: failed to stat database file: %w", op, err)
	}

	return nil
}

func (s *Storage) CleanDb(ctx context.Context) error {
	const op = "storage.sqlite.CleanDb"

	if err := s.queries.DeleteAllCommands(ctx); err != nil {
		return fmt.Errorf("%s: failed to delete all commands: %w", op, err)
	}

	if err := s.queries.DeleteAllParams(ctx); err != nil {
		return fmt.Errorf("%s: failed to delete all parameters: %w", op, err)
	}

	return nil
}
