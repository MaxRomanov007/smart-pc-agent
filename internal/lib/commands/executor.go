package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"smart-pc-desktop-client/internal/lib/logger/sl"

	"github.com/gorilla/websocket"
)

const logUrl = "http://localhost:9080/mqtt/pc/hello/command/log"

type CommandFunc func(context.Context, *Message) error

type Executor struct {
	commands       map[string]CommandFunc
	defaultCommand CommandFunc
}

func NewExecutor() *Executor {
	return &Executor{
		commands:       make(map[string]CommandFunc),
		defaultCommand: nil,
	}
}

func (e *Executor) Set(name string, command CommandFunc) {
	e.commands[name] = command
}

func (e *Executor) SetDefault(command CommandFunc) {
	e.defaultCommand = command
}

func (e *Executor) Start(ctx context.Context, l *slog.Logger, opts *StartOptions) error {
	const op = "lib.commands.executor.Start"

	log := l.With(sl.Op(op))

	if err := opts.check(); err != nil {
		return fmt.Errorf("%s: options validate failed: %w", op, err)
	}

	failingsCount := 0

	var err error
	for failingsCount <= opts.ReconnectAttempts {
		token, authErr := opts.Auth.Token(ctx)
		if authErr == nil {
			log.Debug("listening channel")

			err = e.listen(ctx, log, opts.urlWithToken(token), opts.MessageType, token)
			if err == nil || isConnectionClosedError(err) {
				failingsCount = 0
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("%s: context canceled: %w", op, err)
			}
		}
		if authErr != nil {
			log.Warn("failed to get token", sl.Err(authErr))
		}
		if err != nil {
			log.Warn("failed to listen channel", sl.Err(err))
		}

		failingsCount++

		if failingsCount <= opts.ReconnectAttempts {
			log.Warn("reconnecting", slog.Int("attempt", failingsCount))
			time.Sleep(opts.ReconnectDelay)
		}
	}
	if err != nil {
		return fmt.Errorf("%s: failed to start executor: %w", op, err)
	}

	return nil
}

func isConnectionClosedError(err error) bool {
	return errors.Is(err, websocket.ErrCloseSent) ||
		websocket.IsCloseError(
			err,
			websocket.CloseAbnormalClosure,
			websocket.CloseNoStatusReceived,
		) ||
		strings.Contains(err.Error(), "closed network connection")
}

func isSyntaxError(err error) bool {
	var jsonErr *json.SyntaxError
	return errors.As(err, &jsonErr)
}

func (e *Executor) listen(ctx context.Context, l *slog.Logger, url, messageType, token string) error {
	const op = "lib.commands.executor.listen"

	log := l.With(sl.Op(op))

	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to dial websocket: %w", op, err)
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Error("failed to close connection with websocket")
		}
	}()

	for {
		msg := new(Message)
		err := conn.ReadJSON(msg)
		if err != nil {
			if isConnectionClosedError(err) {
				log.Debug("connection closed", sl.Err(err))
				return err
			}

			if isSyntaxError(err) {
				log.Debug("message syntax error", sl.Err(err))
				continue
			}

			return fmt.Errorf("%s: failed to read message: %w", op, err)
		}

		log.Debug("received message", slog.Any("message", *msg))

		receivedAt := time.Now()

		if msg.Payload.Type != messageType {
			log.Debug("invalid message type, skipping")
			continue
		}

		handler := e.getHandler(msg.Payload.Data.Command)
		if handler == nil {
			log.Debug("handler not found, skipping")
			continue
		}

		err = handler(ctx, msg)

		completedAt := time.Now()

		if err != nil {
			var scriptErr *ScriptError
			if errors.As(err, &scriptErr) {
				log.Debug("script error", sl.Err(scriptErr))

				if err := sendLog(
					logUrl,
					ScriptFailed(msg.Payload.Data.Command, scriptErr, receivedAt, completedAt),
					token,
				); err != nil {
					log.Warn("failed to send script error log", sl.Err(err))
				}
				continue
			}

			log.Warn("failed to handle message", sl.Err(err))
			if err := sendLog(
				logUrl,
				Internal(msg.Payload.Data.Command, receivedAt, completedAt),
				token,
			); err != nil {
				log.Debug("failed to send internal error log", sl.Err(err))
			}
			continue
		}

		if err := sendLog(
			logUrl,
			Done(msg.Payload.Data.Command, receivedAt, completedAt),
			token,
		); err != nil {
			log.Debug("failed to send done log", sl.Err(err))
		}
	}
}

func (e *Executor) getHandler(key string) CommandFunc {
	if handler, ok := e.commands[key]; ok {
		return handler
	}

	return e.defaultCommand
}

type logResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
}

const logResponseStatusOK = 0

func sendLog(url string, resp *Response, token string) error {
	const op = "lib.commands.response.Send"

	data, err := json.Marshal(*resp)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal json: %w", op, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: failed to create request: %w", op, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: failed to send request: %w", op, err)
	}
	defer func() { _ = response.Body.Close() }()

	var result logResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("%s: failed to decode json: %w", op, err)
	}

	if result.Status != logResponseStatusOK {
		return fmt.Errorf("%s: failed to send log: %w", op, errors.New(result.Error))
	}

	return nil
}
