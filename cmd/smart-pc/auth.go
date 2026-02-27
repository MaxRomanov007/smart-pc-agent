package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/authorization"
	"golang.org/x/oauth2"
)

const tokenKey = "token"

func createAuth(log *slog.Logger, queries *dbqueries.Queries) (*authorization.Auth, error) {
	const op = "cmd.smart-pc.createAuth"

	authConfig := &authorization.Config{
		CallbackConfig: authorization.CallbackConfig{
			TTL:          5 * time.Minute,
			IdleTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			ReadTimeout:  5 * time.Second,
			Host:         "127.0.0.1",
		},
		Oauth2Config: &oauth2.Config{
			ClientID: "smart-pc-cmd",
			Scopes: []string{
				"offline",
				"mqtt:pc:state:write",
				"mqtt:pc:command:read",
				"mqtt:pc:log:write",
				"mqtt:pc:status:write",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://kratos:4444/oauth2/auth",
				TokenURL: "http://kratos:4444/oauth2/token",
			},
		},
		LoadToken: func(ctx context.Context) (*oauth2.Token, error) {
			data, err := queries.GetStorageValue(ctx, tokenKey)
			if err != nil {
				return nil, err
			}
			var token oauth2.Token
			if err := json.Unmarshal([]byte(data.Value), &token); err != nil {
				return nil, err
			}
			return &token, nil
		},
		SaveToken: func(ctx context.Context, token *oauth2.Token) error {
			data, err := json.Marshal(token)
			if err != nil {
				return err
			}

			if err := queries.SetStorageValue(ctx, &dbqueries.SetStorageValueParams{
				Key:   tokenKey,
				Value: string(data),
			}); err != nil {
				return err
			}

			return nil
		},
		UserInfoURL: "http://kratos:4444/userinfo",
	}

	auth, err := authorization.Load(context.Background(), authConfig)
	if err != nil {
		log.Debug("failed to load auth", slog.String("error", err.Error()))

		newAuth, err := authorization.New(context.Background(), authConfig)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		auth = newAuth
	}

	return auth, nil
}
