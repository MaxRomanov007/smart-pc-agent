package logger

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/handlers/slogpretty"
)

const (
	envDev   = "dev"
	envDebug = "debug"
	envProd  = "production"
)

func MustSetupLogger(env string) *slog.Logger {
	var log *slog.Logger

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
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo},
		))
	default:
		panic(fmt.Errorf("invalid env type %q. available env types are: %q, %q, %q", env, envDev, envDebug, envProd))
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
