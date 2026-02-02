package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/oauth2"
	"smart-pc-desktop-client/internal/config"
	"smart-pc-desktop-client/internal/lib/authorization"
	"smart-pc-desktop-client/internal/lib/logger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustSetupLogger(cfg.Env)

	log.Debug("debug messages are enabled")

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
			Scopes:   []string{"offline", "mqtt:pc:state:write"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://kratos:4444/oauth2/auth",
				TokenURL: "http://kratos:4444/oauth2/token",
			},
		},
		LoadToken: func(_ context.Context) (*oauth2.Token, error) {
			data, err := os.ReadFile("token.json")
			if err != nil {
				return nil, err
			}
			var token oauth2.Token
			if err := json.Unmarshal(data, &token); err != nil {
				return nil, err
			}
			return &token, nil
		},
		SaveToken: func(_ context.Context, token *oauth2.Token) error {
			data, err := json.Marshal(token)
			if err != nil {
				return err
			}

			if err := os.WriteFile("token.json", data, 0o600); err != nil {
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
			panic(err)
		}

		auth = newAuth
	}

	fmt.Println(auth.TryToken(context.Background()))
}
