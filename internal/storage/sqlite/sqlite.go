package sqlite

import (
	"database/sql"
	"fmt"
	appStorage "smart-pc-agent/internal/storage/sqlite/app-storage"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	AppStorage *appStorage.Storage
}

func New(path string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to connect to database: %w", op, err)
	}
	defer db.Close()

	queries := dbqueries.New(db)

	return &Storage{
		AppStorage: appStorage.New(queries),
	}, nil
}
