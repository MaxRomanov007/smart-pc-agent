-- name: GetCommandParams :many
SELECT *
FROM command_params
WHERE command_id = $command_id;

-- name: CreateOrUpdateCommandParameter :one
INSERT INTO command_params(command_id, name, type)
VALUES ($command_id, $name, $type)
ON CONFLICT(command_id, name)
    DO UPDATE SET type = excluded.type
RETURNING *;

-- name: DeleteCommandParameters :exec
DELETE FROM command_params WHERE command_id = @command_id;

-- name: DeleteCommandParametersExceptNames :exec
DELETE
FROM command_params
WHERE command_id = @command_id
  AND name NOT IN (sqlc.slice('names'));