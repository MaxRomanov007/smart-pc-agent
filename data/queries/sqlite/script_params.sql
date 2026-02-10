-- name: GetScriptParams :many
SELECT * FROM script_params WHERE script_id = $script_id