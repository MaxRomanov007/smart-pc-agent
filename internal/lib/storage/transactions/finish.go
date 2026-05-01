package transactions

import (
	"database/sql"
	"fmt"
)

func Finish(tx *sql.Tx, resultErr *error) {
	const op = "lib.storage.postgres.FinishTx"

	if *resultErr != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			*resultErr = fmt.Errorf(
				"%s: failed to rollback (error: %w), after operation failed (error: %w)",
				op,
				rollbackErr,
				*resultErr,
			)
		}
		return
	}

	commitErr := tx.Commit()
	if commitErr != nil {
		*resultErr = fmt.Errorf("%s: failed to commit transaction: %w", op, commitErr)
	}
}
