-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS script_params
(
    script_id TEXT NOT NULL REFERENCES scripts (id),
    name      VARCHAR(255) NOT NULL,
    type      INTEGER NOT NULL CHECK (type >= 1 AND type <= 3),

    PRIMARY KEY (script_id, name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS script_params;
-- +goose StatementEnd
