package sqlite

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/pressly/goose/v3"
)

type migrationLogger struct {
	*slog.Logger
}

func (m migrationLogger) Fatalf(format string, v ...any) {
	m.Logger.Error(fmt.Sprintf(format, v...))
}

func (m migrationLogger) Printf(format string, v ...any) {
	m.Logger.Info(fmt.Sprintf(format, v...))
}

func migrate(db *sql.DB, log *slog.Logger, migrationsPath string) error {
	const op = "storage.sqlite.migrate"

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("%s: failed to set goose dialect: %w", op, err)
	}
	goose.SetLogger(migrationLogger{log.With(sl.Op(op))})
	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("%s: failed to apply migrations: %w", op, err)
	}

	return nil
}
