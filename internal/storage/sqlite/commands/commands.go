package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"smart-pc-agent/internal/lib/storage/transactions"
	"smart-pc-agent/internal/storage"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"

	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models"
	"github.com/google/uuid"
)

type Storage struct {
	queries *dbqueries.Queries
	db      *sql.DB
}

func New(db *sql.DB) *Storage {
	return &Storage{queries: dbqueries.New(db), db: db}
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

	commandModel, err := mapStorageCommand(command)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to map command: %w", op, err)
	}

	return commandModel, nil
}

func (s Storage) GetCommandScript(ctx context.Context, id uuid.UUID) (string, error) {
	const op = "sqlite.commands.GetCommandScript"

	command, err := s.queries.GetCommandById(ctx, id.String())
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: failed to get command by id: %w", op, err)
	}

	return command.Script, nil
}

func (s Storage) CreateCommand(
	ctx context.Context,
	command models.Command,
) (created models.Command, err error) {
	const op = "sqlite.commands.CreateCommand"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer transactions.Finish(tx, &err)

	queries := dbqueries.New(tx)

	createdCommand, err := queries.CreateCommand(ctx, dbqueries.CreateCommandParams{
		ID:     command.ID.String(),
		Script: command.Script,
	})
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to create command: %w", op, err)
	}

	if command.Parameters == nil {
		commandModel, err := mapStorageCommand(createdCommand)
		if err != nil {
			return models.Command{}, fmt.Errorf(
				"%s: failed to map command on nil parameters: %w",
				op,
				err,
			)
		}

		return commandModel, nil
	}

	for _, param := range command.Parameters {
		_, err := queries.CreateOrUpdateCommandParameter(
			ctx,
			dbqueries.CreateOrUpdateCommandParameterParams{
				CommandID: command.ID.String(),
				Name:      param.Name,
				Type:      param.Type,
			},
		)
		if err != nil {
			return models.Command{}, fmt.Errorf(
				"%s: failed to create command parameter (name: %s): %w",
				op,
				param.Name,
				err,
			)
		}
	}

	commandModel, err := mapStorageCommand(createdCommand)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to map command: %w", op, err)
	}

	return commandModel, nil
}

func (s Storage) DeleteCommand(ctx context.Context, id uuid.UUID) (models.Command, error) {
	const op = "sqlite.commands.GetCommandScript"

	command, err := s.queries.DeleteCommand(ctx, id.String())
	if errors.Is(err, sql.ErrNoRows) {
		return models.Command{}, storage.ErrNotFound
	}
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to delete command by id: %w", op, err)
	}

	if err := s.queries.DeleteCommandParameters(ctx, id.String()); err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to delete command parameters: %w", op, err)
	}

	commandModel, err := mapStorageCommand(command)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to map command: %w", op, err)
	}

	return commandModel, nil
}

func (s Storage) UpdateCommand(
	ctx context.Context,
	command models.Command,
) (updated models.Command, err error) {
	const op = "sqlite.commands.UpdateCommand"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer transactions.Finish(tx, &err)

	queries := dbqueries.New(tx)

	updatedCommand, err := queries.UpdateCommandScript(ctx, dbqueries.UpdateCommandScriptParams{
		Script: command.Script,
		ID:     command.ID.String(),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return models.Command{}, storage.ErrNotFound
	}
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to update command script: %w", op, err)
	}

	command.Script = updatedCommand.Script

	if command.Parameters == nil {
		return command, nil
	}

	newNames := make([]string, len(command.Parameters))
	for i, param := range command.Parameters {
		storageParam, err := queries.CreateOrUpdateCommandParameter(
			ctx,
			dbqueries.CreateOrUpdateCommandParameterParams{
				CommandID: command.ID.String(),
				Name:      param.Name,
				Type:      param.Type,
			},
		)
		if err != nil {
			return models.Command{}, fmt.Errorf(
				"%s: failed to create or update command parameter (name: %s): %w",
				op,
				param.Name,
				err,
			)
		}

		newNames[i] = storageParam.Name

		paramModel, err := mapStorageCommandParam(storageParam)
		if err != nil {
			return models.Command{}, fmt.Errorf(
				"%s: failed to parse command parameter (command_id=%s,name=%s): %w",
				op,
				param.CommandID,
				param.Name,
				err,
			)
		}

		command.Parameters[i] = paramModel
	}

	if err := queries.DeleteCommandParametersExceptNames(
		ctx,
		dbqueries.DeleteCommandParametersExceptNamesParams{
			CommandID: command.ID.String(),
			Names:     newNames,
		},
	); err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to delete command parameters: %w", op, err)
	}

	return command, nil
}

func mapStorageCommand(command dbqueries.Command) (models.Command, error) {
	const op = "sqlite.commands.mapStorageCommand"

	commandUUID, err := uuid.Parse(command.ID)
	if err != nil {
		return models.Command{}, fmt.Errorf(
			"%s: failed to parse command uuid: %w",
			op,
			err,
		)
	}

	return models.Command{
		ID:     &commandUUID,
		Script: command.Script,
	}, nil
}

func mapStorageCommandParam(param dbqueries.CommandParam) (models.CommandParameter, error) {
	const op = "sqlite.commands.mapStorageCommandParam"

	commandUUID, err := uuid.Parse(param.CommandID)
	if err != nil {
		return models.CommandParameter{}, fmt.Errorf(
			"%s: failed to parse command uuid: %w",
			op,
			err,
		)
	}

	return models.CommandParameter{
		CommandID: commandUUID,
		Name:      param.Name,
		Type:      param.Type,
	}, nil
}
