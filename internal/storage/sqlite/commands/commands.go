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

func (s Storage) CreateCommand(
	ctx context.Context,
	command models.Command,
) (created models.Command, err error) {
	const op = "sqlite.commands.CreateCommand"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf(
					"%s: failed to rollback (error: %w), after operation failed (error: %w)",
					op,
					rollbackErr,
					err,
				)
			}
			return
		}

		commitErr := tx.Commit()
		if commitErr != nil {
			err = fmt.Errorf("%s: failed to commit transaction: %w", op, commitErr)
		}
	}()

	queries := dbqueries.New(tx)

	createdCommand, err := queries.CreateCommand(ctx, dbqueries.CreateCommandParams{
		ID:     command.ID,
		Script: command.Script,
	})
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to create command: %w", op, err)
	}

	if command.Parameters == nil {
		return mapStorageCommand(createdCommand), nil
	}

	for _, param := range command.Parameters {
		_, err := queries.CreateOrUpdateCommandParameter(
			ctx,
			dbqueries.CreateOrUpdateCommandParameterParams{
				CommandID: command.ID,
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

	return mapStorageCommand(createdCommand), nil
}

func (s Storage) DeleteCommand(ctx context.Context, id string) (models.Command, error) {
	const op = "sqlite.commands.GetCommandScript"

	command, err := s.queries.DeleteCommand(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Command{}, storage.ErrNotFound
	}
	if err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to delete command by id: %w", op, err)
	}

	if err := s.queries.DeleteCommandParameters(ctx, id); err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to delete command parameters: %w", op, err)
	}

	return mapStorageCommand(command), nil
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

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf(
					"%s: failed to rollback (error: %w), after operation failed (error: %w)",
					op,
					rollbackErr,
					err,
				)
			}
			return
		}

		commitErr := tx.Commit()
		if commitErr != nil {
			err = fmt.Errorf("%s: failed to commit transaction: %w", op, commitErr)
		}
	}()

	queries := dbqueries.New(tx)

	updatedCommand, err := queries.UpdateCommandScript(ctx, dbqueries.UpdateCommandScriptParams{
		Script: command.Script,
		ID:     command.ID,
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
				CommandID: command.ID,
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
		command.Parameters[i] = mapStorageCommandParam(storageParam)
	}

	if err := queries.DeleteCommandParametersExceptNames(
		ctx,
		dbqueries.DeleteCommandParametersExceptNamesParams{
			CommandID: command.ID,
			Names:     newNames,
		},
	); err != nil {
		return models.Command{}, fmt.Errorf("%s: failed to delete command parameters: %w", op, err)
	}

	return command, nil
}

func mapStorageCommand(command dbqueries.Command) models.Command {
	return models.Command{
		ID:     command.ID,
		Script: command.Script,
	}
}

func mapStorageCommandParam(param dbqueries.CommandParam) models.CommandParameter {
	return models.CommandParameter{
		CommandID: param.CommandID,
		Name:      param.Name,
		Type:      param.Type,
	}
}
