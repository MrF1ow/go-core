-- name: GetAllSettings :many
SELECT key, value, category, updated_at
FROM system_settings
ORDER BY category ASC, key ASC;

-- name: GetSettingsByCategory :many
SELECT key, value, category, updated_at
FROM system_settings
WHERE category = $1
ORDER BY key ASC;

-- name: GetSettingByKey :one
SELECT key, value, category, updated_at
FROM system_settings
WHERE key = $1;

-- name: UpsertSetting :exec
INSERT INTO system_settings (key, value, category, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW();

-- name: DeleteSetting :exec
DELETE FROM system_settings
WHERE key = $1;
