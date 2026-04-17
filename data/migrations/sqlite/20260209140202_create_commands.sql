-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS commands
(
    id     TEXT PRIMARY KEY,
    script VARCHAR(8192) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS commands;
-- +goose StatementEnd
