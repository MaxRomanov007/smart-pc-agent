package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/lib/logger"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"

	"github.com/MaxRomanov007/smart-pc-go-lib/authorization"
	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustSetupLogger(cfg.Env)

	log.Debug("debug messages are enabled")

	db, err := sql.Open("sqlite3", "./data/database/db.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	queries := dbqueries.New(db)

	auth, err := createAuth(log, queries)
	if err != nil {
		panic(err)
	}

	executor, err := commands.NewExecutor(
		"http://localhost:9080/mqtt/pc/hello/command/log",
		"desktop-command-log",
	)
	if err != nil {
		panic(err)
	}

	executor.Set("hello", func(ctx context.Context, message *commands.Message) error {
		log.Debug("hello", slog.Any("message", message))
		return nil
	})

	executor.Start(context.Background(), log, &commands.StartOptions{
		Auth:              auth,
		URL:               "ws://localhost:9080/mqtt/pc/hello/command",
		MessageType:       "command",
		ReconnectDelay:    time.Second * 3,
		ReconnectAttempts: 5,
	})
}

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

			if err := queries.SetStorageValue(ctx, dbqueries.SetStorageValueParams{
				Key:   tokenKey,
				Value: string(data),
			}); err != nil {
				return err
			}

			return nil
		},
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
