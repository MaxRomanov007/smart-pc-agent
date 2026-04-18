package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"smart-pc-agent/internal/domain/models"
	"smart-pc-agent/internal/storage"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"
)

type Storage struct {
	queries *dbqueries.Queries
}

func New(queries *dbqueries.Queries) *Storage {
	return &Storage{queries}
}

func (s Storage) GetCommandById(ctx context.Context, id string) (models.Command, error) {
	const op = "sqlite.commands.GetCommandById"

	command, err := s.queries.GetCommandById(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Command{}, storage.ErrNotFound
	}
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to get command by id: %w", op, err)
	}

	return mapStorageCommand(command), nil
}

func (s Storage) GetCommandScript(ctx context.Context, id string) (string, error) {
	const op = "sqlite.commands.GetCommandScript"

	command, err := s.queries.GetCommandById(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: failed to get command by id: %w", op, err)
	}

	return command.Script, nil
}

func mapStorageCommand(command dbqueries.Command) models.Command {
	return models.Command{
		ID:     command.ID,
		Script: command.Script,
	}
}
