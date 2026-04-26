package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

const script = `
CREATE TABLE IF NOT EXISTS app_storage
(
    key   VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS commands
(
    id     TEXT PRIMARY KEY,
    script VARCHAR(8192) NOT NULL
);
CREATE TABLE IF NOT EXISTS command_params
(
    command_id TEXT         NOT NULL REFERENCES commands (id),
    name       VARCHAR(255) NOT NULL,
    type       SMALLINT     NOT NULL CHECK (type >= 1 AND type <= 3),

    PRIMARY KEY (command_id, name)
);`

func migrate(db *sql.DB, ctx context.Context) (err error) {
	const op = "storage.sqlite.migrate"

	if _, err := db.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("%s: failed to execute script: %w", op, err)
	}

	return nil
}
