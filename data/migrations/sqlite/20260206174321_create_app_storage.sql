-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS app_storage
(
    key   VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS app_storage;
-- +goose StatementEnd
