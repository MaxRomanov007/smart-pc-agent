package appStorage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"smart-pc-agent/internal/storage"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"

	"golang.org/x/oauth2"
)

const (
	authTokenKey = "auth_token"
)

type Storage struct {
	queries *dbqueries.Queries
}

func New(queries *dbqueries.Queries) *Storage {
	return &Storage{queries}
}

func (s Storage) GetAuthToken(ctx context.Context) (*oauth2.Token, error) {
	const op = "sqlite.app-storage.GetAuthToken"

	data, err := s.queries.GetStorageValue(ctx, authTokenKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get token: %w", op, err)
	}

	token := new(oauth2.Token)
	if err := json.Unmarshal([]byte(data.Value), token); err != nil {
		return nil, fmt.Errorf("%s: failed to unmarshal token: %w", op, err)
	}
	return token, nil
}

func (s Storage) SetAuthToken(ctx context.Context, token *oauth2.Token) error {
	const op = "sqlite.app-storage.SetAuthToken"

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal token: %w", op, err)
	}

	if err := s.queries.SetStorageValue(ctx, &dbqueries.SetStorageValueParams{
		Key:   authTokenKey,
		Value: string(data),
	}); err != nil {
		return fmt.Errorf("%s: failed to set auth token: %w", op, err)
	}

	return nil
}
