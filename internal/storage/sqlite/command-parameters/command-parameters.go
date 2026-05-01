package commandParameters

import (
	"context"
	"fmt"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"

	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models"
	"github.com/google/uuid"
)

type Storage struct {
	queries *dbqueries.Queries
}

func New(queries *dbqueries.Queries) *Storage {
	return &Storage{queries}
}

func (s Storage) GetCommandParams(
	ctx context.Context,
	commandId uuid.UUID,
) ([]models.CommandParameter, error) {
	const op = "sqlite.command-parameters.GetCommandParams"

	params, err := s.queries.GetCommandParams(ctx, commandId.String())
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get command params: %w", op, err)
	}

	modelParams, err := mapStorageParams(params)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to map storage params: %w", op, err)
	}

	return modelParams, nil
}

func mapStorageParams(raw []dbqueries.CommandParam) ([]models.CommandParameter, error) {
	const op = "sqlite.command-parameters.mapStorageParams"

	params := make([]models.CommandParameter, len(raw))
	for i, param := range raw {
		model, err := mapStorageParam(param)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: failed to parse command parameter (command_id=%s,name=%s): %w",
				op,
				param.CommandID,
				param.Name,
				err,
			)
		}
		params[i] = model
	}
	return params, nil
}

func mapStorageParam(param dbqueries.CommandParam) (models.CommandParameter, error) {
	const op = "sqlite.command-parameters.mapStorageParam"

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
