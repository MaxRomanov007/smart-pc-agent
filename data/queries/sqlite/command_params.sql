-- name: GetCommandParams :many
SELECT *
FROM command_params
WHERE command_id = $command_id;

-- name: CreateCommandParameter :one
INSERT INTO command_params(command_id, name, type)
VALUES ($command_id, $name, $type)
RETURNING *;

-- name: DeleteCommandParameters :exec
DELETE FROM command_params WHERE command_id = @command_id