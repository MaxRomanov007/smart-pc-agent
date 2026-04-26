package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"smart-pc-agent/data/schema"
)

func migrate(db *sql.DB, ctx context.Context) (err error) {
	const op = "storage.sqlite.migrate"

	if _, err := db.ExecContext(ctx, schema.GetSchemaScript()); err != nil {
		return fmt.Errorf("%s: failed to execute script: %w", op, err)
	}

	return nil
}
