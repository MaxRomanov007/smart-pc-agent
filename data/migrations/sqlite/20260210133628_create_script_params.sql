-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS command_params
(
    command_id TEXT NOT NULL REFERENCES commands (id),
    name      VARCHAR(255) NOT NULL,
    type      INTEGER NOT NULL CHECK (type >= 1 AND type <= 3),

    PRIMARY KEY (command_id, name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS command_params;
-- +goose StatementEnd
