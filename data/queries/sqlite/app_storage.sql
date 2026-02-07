-- name: SetStorageValue :exec
INSERT OR REPLACE INTO app_storage(key, value) VALUES ($key, $value);

-- name: GetStorageValue :one
SELECT * FROM app_storage WHERE key = $key;

-- name: DeleteStorageValue :exec
DELETE FROM app_storage WHERE key = $key;