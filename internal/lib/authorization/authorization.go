package authorization

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

// Auth manages OAuth2 authentication and token lifecycle.
// It provides thread-safe access to tokens and handles automatic token refresh.
type Auth struct {
	cfg        *Config       // Configuration for OAuth2 flow
	OAuthToken *oauth2.Token // Current OAuth2 token
	tokenMux   sync.Mutex    // Mutex for thread-safe token access
}

// New creates a new Auth instance by performing a complete OAuth2 authorization flow.
// This initiates browser-based authentication and exchanges the authorization code for tokens.
func New(ctx context.Context, cfg *Config) (*Auth, error) {
	const op = "lib.authorization.authorization.New"

	token, err := cfg.acquireNewToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get token: %w", op, err)
	}

	return &Auth{
		cfg:        cfg,
		OAuthToken: token,
	}, nil
}

// Load creates an Auth instance using a previously saved token.
// The token is loaded via the LoadToken function in Config and refreshed if expired.
func Load(ctx context.Context, cfg *Config) (*Auth, error) {
	const op = "lib.authorization.authorization.Load"

	token, err := cfg.loadToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load token: %w", op, err)
	}

	return &Auth{
		cfg:        cfg,
		OAuthToken: token,
	}, nil
}

// Token retrieves the current access token, refreshing it if necessary.
// This method blocks if another goroutine is currently accessing the token.
// Returns the access token string or an error if token is unavailable or invalid.
func (a *Auth) Token(ctx context.Context) (string, error) {
	a.tokenMux.Lock()
	defer a.tokenMux.Unlock()

	return a.tokenDangerously(ctx)
}

// TryToken attempts to retrieve the current access token without blocking.
// Returns ErrTokenLocked if the token is currently locked by another goroutine.
// Useful for non-blocking token access in performance-critical paths.
func (a *Auth) TryToken(ctx context.Context) (string, error) {
	if !a.tokenMux.TryLock() {
		return "", ErrTokenLocked
	}
	defer a.tokenMux.Unlock()

	return a.tokenDangerously(ctx)
}

// tokenDangerously retrieves or refreshes the token without locking.
// This is an internal method and must be called with the tokenMux already locked.
// It handles token validation and automatic refresh using the OAuth2 token source.
func (a *Auth) tokenDangerously(ctx context.Context) (string, error) {
	const op = "lib.authorization.authorization.tokenDangerously"

	if a.OAuthToken == nil {
		return "", ErrNoToken
	}

	if !a.OAuthToken.Valid() {
		token, err := a.cfg.Oauth2Config.TokenSource(ctx, a.OAuthToken).Token()
		if err != nil {
			return "", fmt.Errorf("%s: failed to update invalid token: %w", op, err)
		}

		a.OAuthToken = token
	}

	return a.OAuthToken.AccessToken, nil
}
