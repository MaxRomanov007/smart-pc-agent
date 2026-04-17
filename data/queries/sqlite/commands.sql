-- name: GetCommandById :one
SELECT *
FROM commands
WHERE id = $id;