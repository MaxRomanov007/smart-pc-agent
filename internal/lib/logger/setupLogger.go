package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/handlers/slogpretty"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

const (
	envDev   = "dev"
	envDebug = "debug"
	envProd  = "production"
)

func MustSetupLogger(ctx context.Context, env string) *slog.Logger {
	var log *slog.Logger

	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		panic(err)
	}

	go func() {
		<-ctx.Done()
		if err := logFile.Close(); err != nil {
			log.Error("failed to close log file", sl.Err(err))
		}
	}()

	switch env {
	case envDev:
		log = setupPrettySlog()
	case envDebug:
		log = slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug},
		))
	case envProd:
		log = slog.New(slog.NewJSONHandler(
			logFile,
			&slog.HandlerOptions{Level: slog.LevelInfo},
		))
	default:
		panic(
			fmt.Errorf(
				"invalid env type %q. available env types are: %q, %q, %q",
				env,
				envDev,
				envDebug,
				envProd,
			),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
