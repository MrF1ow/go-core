-- name: CreateTrustedDevice :exec
INSERT INTO trusted_devices (id, user_id, app_id, token_hash, name, user_agent, ip_address, last_used_at, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: FindTrustedDeviceByTokenHash :one
SELECT id, user_id, app_id, token_hash, name, user_agent, ip_address, last_used_at, expires_at, created_at
FROM trusted_devices
WHERE token_hash = $1;

-- name: FindTrustedDevicesByUserAndApp :many
SELECT id, user_id, app_id, token_hash, name, user_agent, ip_address, last_used_at, expires_at, created_at
FROM trusted_devices
WHERE user_id = $1 AND app_id = $2 AND expires_at > NOW()
ORDER BY last_used_at DESC;

-- name: FindTrustedDeviceByID :one
SELECT id, user_id, app_id, token_hash, name, user_agent, ip_address, last_used_at, expires_at, created_at
FROM trusted_devices
WHERE id = $1;

-- name: TouchTrustedDeviceLastUsed :exec
UPDATE trusted_devices
SET last_used_at = NOW()
WHERE id = $1;

-- name: DeleteTrustedDeviceByID :exec
DELETE FROM trusted_devices
WHERE id = $1;

-- name: DeleteAllTrustedDevicesForUser :exec
DELETE FROM trusted_devices
WHERE user_id = $1 AND app_id = $2;

-- name: DeleteTrustedDevicesByUserAppAndUserAgent :exec
DELETE FROM trusted_devices
WHERE user_id = $1 AND app_id = $2 AND user_agent = $3;

-- name: DeleteExpiredTrustedDevices :execrows
DELETE FROM trusted_devices
WHERE expires_at < NOW();

-- name: CountTrustedDevicesByUserAndApp :one
SELECT COUNT(*) FROM trusted_devices
WHERE user_id = $1 AND app_id = $2 AND expires_at > NOW();

-- name: CountAllActiveTrustedDevices :one
SELECT COUNT(*) FROM trusted_devices
WHERE expires_at > NOW();

-- name: DeleteAllTrustedDevicesForUserAllApps :exec
DELETE FROM trusted_devices
WHERE user_id = $1;

-- name: FindAllTrustedDevicesForUser :many
SELECT id, user_id, app_id, token_hash, name, user_agent, ip_address, last_used_at, expires_at, created_at
FROM trusted_devices
WHERE user_id = $1 AND expires_at > NOW()
ORDER BY last_used_at DESC;
