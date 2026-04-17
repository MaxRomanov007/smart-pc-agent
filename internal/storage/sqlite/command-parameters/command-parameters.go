package commandParameters

import (
	"context"
	"fmt"
	"smart-pc-agent/internal/domain/models"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"
)

type Storage struct {
	queries *dbqueries.Queries
}

func New(queries *dbqueries.Queries) *Storage {
	return &Storage{queries}
}

func (s Storage) GetCommandParams(
	ctx context.Context,
	commandId string,
) ([]models.CommandParameter, error) {
	const op = "sqlite.command-parameters.GetCommandParams"

	params, err := s.queries.GetCommandParams(ctx, commandId)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get command params: %w", op, err)
	}

	return mapStorageParams(params), nil
}

func mapStorageParams(raw []dbqueries.CommandParam) []models.CommandParameter {
	params := make([]models.CommandParameter, len(raw))
	for i, param := range raw {
		params[i] = mapStorageParam(param)
	}
	return params
}

func mapStorageParam(param dbqueries.CommandParam) models.CommandParameter {
	return models.CommandParameter{
		CommandID: param.CommandID,
		Name:      param.Name,
		Type:      param.Type,
	}
}
