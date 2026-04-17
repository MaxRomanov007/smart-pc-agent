-- name: GetCommandParams :many
SELECT * FROM command_params WHERE command_id = $command_id