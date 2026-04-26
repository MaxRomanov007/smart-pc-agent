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
);