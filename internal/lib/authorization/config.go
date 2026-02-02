package authorization

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"smart-pc-desktop-client/internal/lib/cross-platform/browser"
)

type (
	// LoadTokenFunc is type of function to load saved tokens
	LoadTokenFunc func(context.Context) (*oauth2.Token, error)
	// SaveTokenFunc is type of function to save token
	SaveTokenFunc func(context.Context, *oauth2.Token) error
	// CallbackConfig is structure for configuring an HTTP server,
	// deployed to implement OAuth2 flow
	CallbackConfig struct {
		Host         string        // Host for callback server (e.g., "127.0.0.1")
		TTL          time.Duration // Maximum time to wait for callback
		ReadTimeout  time.Duration // HTTP server read timeout
		WriteTimeout time.Duration // HTTP server write timeout
		IdleTimeout  time.Duration // HTTP server idle timeout
	}
)

// Config holds configuration for OAuth2 authorization flow.
// It includes OAuth2 configuration, token loading function, and server settings.
type Config struct {
	Oauth2Config   *oauth2.Config // OAuth2 client configuration
	LoadToken      LoadTokenFunc  // Function to load saved tokens
	SaveToken      SaveTokenFunc  // Function to save token
	CallbackConfig CallbackConfig
}

// acquireNewToken performs a complete OAuth2 authorization flow to obtain a new token.
// This includes PKCE challenge generation, browser redirection, and authorization code exchange.
func (cfg *Config) acquireNewToken(ctx context.Context) (*oauth2.Token, error) {
	const op = "lib.authorization.config.token"

	err := cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("%s: config validation failed: %w", op, err)
	}

	token, err := cfg.authorizeUsingBrowser(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to authorize using browser: %w", op, err)
	}

	if err := cfg.saveTokenIfNeeded(ctx, token); err != nil {
		return nil, fmt.Errorf("%s: failed to save token: %w", op, err)
	}

	return token, nil
}

// loadToken loads a previously saved token and refreshes it if expired.
// Returns an error if LoadToken function is not defined or token loading fails.
func (cfg *Config) loadToken(ctx context.Context) (*oauth2.Token, error) {
	const op = "lib.authorization.config.loadToken"

	if cfg.LoadToken == nil {
		return nil, fmt.Errorf("%s: no LoadToken function defined", op)
	}

	loadedToken, err := cfg.LoadToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to load token: %w", op, err)
	}

	if !loadedToken.Valid() {
		newToken, err := cfg.Oauth2Config.TokenSource(ctx, loadedToken).Token()
		if err != nil {
			return nil, fmt.Errorf("%s: failed to update loaded token: %w", op, err)
		}
		loadedToken = newToken
	}

	if err := cfg.saveTokenIfNeeded(ctx, loadedToken); err != nil {
		return nil, fmt.Errorf("%s: failed to save loaded token: %w", op, err)
	}

	return loadedToken, nil
}

// authorizeUsingBrowser performs browser-based OAuth2 authorization with PKCE.
// Generates PKCE parameters, starts a callback server, opens browser for authentication,
// and exchanges the authorization code for tokens.
func (cfg *Config) authorizeUsingBrowser(ctx context.Context) (*oauth2.Token, error) {
	const op = "lib.authorization.config.authorizeUsingBrowser"

	params, err := generatePKCEParams()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to generate params: %w", op, err)
	}

	cfg.Oauth2Config.RedirectURL = fmt.Sprintf(
		"http://%s:%d/callback",
		cfg.CallbackConfig.Host,
		params.port,
	)

	authCodeURL := cfg.generateAuthCodeUrl(params.state, params.challenge)

	if err := browser.OpenContext(ctx, authCodeURL); err != nil {
		return nil, fmt.Errorf("%s: failed to open auth code url in browser: %w", op, err)
	}

	code, err := cfg.getCallbackCodeWithTimeout(
		ctx,
		params.state,
		cfg.CallbackConfig.Host,
		params.port,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get callback code: %w", op, err)
	}

	token, err := cfg.Oauth2Config.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", params.verifier),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to exchange token: %w", op, err)
	}

	return token, nil
}

// generateAuthCodeUrl creates the OAuth2 authorization URL with PKCE parameters.
// Includes state for CSRF protection and code challenge for PKCE.
func (cfg *Config) generateAuthCodeUrl(state, challenge string) string {
	return cfg.Oauth2Config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

// getCallbackCodeWithTimeout starts a callback server with a timeout context.
// Wraps getCallbackCode with context timeout based on CallbackConfig.TTL.
func (cfg *Config) getCallbackCodeWithTimeout(
	ctx context.Context,
	state string,
	host string,
	port int,
) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.CallbackConfig.TTL)
	defer cancel()

	return cfg.getCallbackCode(timeoutCtx, state, host, port)
}

// getCallbackCode starts an HTTP server to receive the OAuth2 callback.
// The server listens on the specified host and port, validates the state parameter,
// extracts the authorization code, and sends it through a channel.
// Returns the authorization code or an error if timeout or validation fails.
func (cfg *Config) getCallbackCode(
	ctx context.Context,
	state string,
	host string,
	port int,
) (string, error) {
	const op = "lib.authorization.config.startCallbackServer"

	codeChan := make(chan string, 1)
	defer close(codeChan)
	errChan := make(chan error, 1)
	defer close(errChan)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		ReadTimeout:  cfg.CallbackConfig.ReadTimeout,
		IdleTimeout:  cfg.CallbackConfig.IdleTimeout,
		WriteTimeout: cfg.CallbackConfig.WriteTimeout,
		Handler:      newCallbackHandler(state, codeChan, errChan),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("%s: failed to serve callback server: %w", op, err)
		}
	}()

	var code string
	var err error
	select {
	case code = <-codeChan:
	case err = <-errChan:
	case <-ctx.Done():
		err = ctx.Err()
	}

	_ = server.Shutdown(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: failed to get callback code from server: %w", op, err)
	}

	return code, nil
}

// validate checks if all required configuration fields are set.
// Returns an error if any required field is missing or invalid.
func (cfg *Config) validate() error {
	var errs []error

	if cfg.Oauth2Config.ClientID == "" {
		errs = append(errs, errors.New("missing client id"))
	}
	if cfg.Oauth2Config.Endpoint.AuthURL == "" {
		errs = append(errs, errors.New("missing auth url"))
	}
	if cfg.Oauth2Config.Endpoint.TokenURL == "" {
		errs = append(errs, errors.New("missing token url"))
	}
	if cfg.CallbackConfig.Host == "" {
		errs = append(errs, errors.New("missing redirect host"))
	}
	if cfg.CallbackConfig.TTL == 0 {
		errs = append(errs, errors.New("missing callback server ttl"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// refreshToken forces a refresh token and saves the result
func (cfg *Config) refreshToken(
	ctx context.Context,
	tokenToRefresh *oauth2.Token,
) (*oauth2.Token, error) {
	const op = "lib.authorization.config.refreshToken"

	token, err := cfg.Oauth2Config.TokenSource(ctx, tokenToRefresh).Token()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to refresh token: %w", op, err)
	}

	if err := cfg.saveTokenIfNeeded(ctx, token); err != nil {
		return nil, fmt.Errorf("%s: failed to save token: %w", op, err)
	}

	return token, nil
}

// saveTokenIfNeeded saves the token if the SaveToken function is present
func (cfg *Config) saveTokenIfNeeded(ctx context.Context, token *oauth2.Token) error {
	const op = "lib.authorization.config.saveIfNeeded"

	if cfg.SaveToken == nil {
		return nil
	}

	if err := cfg.SaveToken(ctx, token); err != nil {
		return fmt.Errorf("%s: save failed: %w", op, err)
	}

	return nil
}
