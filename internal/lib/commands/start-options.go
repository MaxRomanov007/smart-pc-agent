package commands

import (
	"errors"
	"net/url"
	"time"

	"smart-pc-desktop-client/internal/lib/authorization"
)

type StartOptions struct {
	Auth              *authorization.Auth
	URL               string
	MessageType       string
	ReconnectDelay    time.Duration
	ReconnectAttempts int
}

func (o *StartOptions) check() error {
	errs := make([]error, 0, 3)

	if o.Auth == nil {
		errs = append(errs, errors.New("authorization required"))
	}
	if o.URL == "" {
		errs = append(errs, errors.New("url required"))
	}
	if o.MessageType == "" {
		errs = append(errs, errors.New("message type required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (o *StartOptions) urlWithToken(token string) string {
	return o.URL + "?token=" + url.QueryEscape(token)
}
