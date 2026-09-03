-- ============================================================================
-- Activity log queries
-- ============================================================================

-- name: GetActivityLogByID :one
SELECT id, app_id, user_id, event_type, timestamp, ip_address, user_agent, details, severity, expires_at, is_anomaly
FROM activity_logs
WHERE id = $1;

-- name: CountUserActivityLogs :one
SELECT COUNT(*)
FROM activity_logs
WHERE user_id = @user_id
  AND (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR timestamp >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR timestamp < sqlc.narg('end_date'));

-- name: ListUserActivityLogs :many
SELECT id, app_id, user_id, event_type, timestamp, ip_address, user_agent, details, severity, expires_at, is_anomaly
FROM activity_logs
WHERE user_id = @user_id
  AND (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR timestamp >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR timestamp < sqlc.narg('end_date'))
ORDER BY timestamp DESC
OFFSET @offset_val LIMIT @limit_val;

-- name: CountAllActivityLogs :one
SELECT COUNT(*)
FROM activity_logs
WHERE (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR timestamp >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR timestamp < sqlc.narg('end_date'))
  AND (sqlc.narg('app_id')::uuid IS NULL OR app_id = sqlc.narg('app_id')::uuid);

-- name: ListAllActivityLogs :many
SELECT id, app_id, user_id, event_type, timestamp, ip_address, user_agent, details, severity, expires_at, is_anomaly
FROM activity_logs
WHERE (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR timestamp >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR timestamp < sqlc.narg('end_date'))
  AND (sqlc.narg('app_id')::uuid IS NULL OR app_id = sqlc.narg('app_id')::uuid)
ORDER BY timestamp DESC
OFFSET @offset_val LIMIT @limit_val;

-- name: ExportUserActivityLogs :many
SELECT id, app_id, user_id, event_type, timestamp, ip_address, user_agent, details, severity, expires_at, is_anomaly
FROM activity_logs
WHERE user_id = @user_id
  AND (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR timestamp >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR timestamp < sqlc.narg('end_date'))
ORDER BY timestamp DESC
LIMIT @limit_val;

-- name: ExportAllActivityLogs :many
SELECT id, app_id, user_id, event_type, timestamp, ip_address, user_agent, details, severity, expires_at, is_anomaly
FROM activity_logs
WHERE (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR timestamp >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR timestamp < sqlc.narg('end_date'))
  AND (sqlc.narg('app_id')::uuid IS NULL OR app_id = sqlc.narg('app_id')::uuid)
ORDER BY timestamp DESC
LIMIT @limit_val;

-- name: CreateActivityLog :exec
INSERT INTO activity_logs (app_id, user_id, event_type, timestamp, ip_address, user_agent, details, severity, expires_at, is_anomaly)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
