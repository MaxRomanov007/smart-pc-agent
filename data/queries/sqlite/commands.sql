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