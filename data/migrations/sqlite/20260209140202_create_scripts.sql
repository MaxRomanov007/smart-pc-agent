-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS scripts
(
    id          TEXT PRIMARY KEY,
    name        VARCHAR(255)  NOT NULL UNIQUE CHECK (LENGTH(name) >= 3),
    description VARCHAR(512),
    text        VARCHAR(8192) NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scripts;
-- +goose StatementEnd
