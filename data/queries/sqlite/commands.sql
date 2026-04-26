-- name: GetCommandById :one
SELECT *
FROM commands
WHERE id = $id;

-- name: CreateCommand :one
INSERT INTO commands(id, script)
VALUES (?, ?)
RETURNING *;

-- name: DeleteCommand :one
DELETE
FROM commands
WHERE id = @id
RETURNING *;

-- name: UpdateCommandScript :one
UPDATE commands
SET script = @script
WHERE id = @id
RETURNING *;

-- name: DeleteAllCommands :exec
-- noinspection SqlWithoutWhere
DELETE
FROM commands