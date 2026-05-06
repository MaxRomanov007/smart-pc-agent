package auth

import (
	"context"
	"errors"
	"fmt"
	"smart-pc-agent/internal/config"
	"smart-pc-agent/internal/storage"

	"github.com/MaxRomanov007/smart-pc-go-lib/authorization"
	authBrowser "github.com/MaxRomanov007/smart-pc-go-lib/authorization/browser"
	"golang.org/x/oauth2"
)

type TokenGetter interface {
	GetAuthToken(ctx context.Context) (*oauth2.Token, error)
}

type TokenSetter interface {
	SetAuthToken(ctx context.Context, token *oauth2.Token) error
}

func New(
	ctx context.Context,
	cfg config.Auth,
	getter TokenGetter,
	setter TokenSetter,
) (*authorization.Auth, error) {
	const op = "auth.New"

	authConfig := &authorization.Config{
		Oauth2Config: &oauth2.Config{
			ClientID: cfg.Oauth2.ClientID,
			Scopes:   cfg.Oauth2.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  cfg.Oauth2.Endpoint.AuthURL,
				TokenURL: cfg.Oauth2.Endpoint.TokenURL,
			},
		},
		LoadToken: func(inner context.Context) (*oauth2.Token, error) {
			return getter.GetAuthToken(inner)
		},
		SaveToken: func(inner context.Context, token *oauth2.Token) error {
			return setter.SetAuthToken(inner, token)
		},
		UserInfoURL: cfg.UserinfoURL,
	}

	auth, err := authorization.Load(ctx, authConfig)
	if errors.Is(err, storage.ErrNotFound) {
		auth, err = authBrowser.Authorize(ctx, authConfig, authBrowser.CallbackConfig(cfg.Callback))
		if err != nil {
			return nil, fmt.Errorf("%s: failed to authorize: %w", op, err)
		}
		return auth, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load token: %w", op, err)
	}

	return auth, nil
}
