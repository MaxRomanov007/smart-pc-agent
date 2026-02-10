-- name: GetScriptById :one
SELECT *
FROM scripts
WHERE id = $id;