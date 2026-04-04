-- ============================================================================
-- Endpoint operations
-- ============================================================================

-- name: CreateWebhookEndpoint :exec
INSERT INTO webhook_endpoints (id, app_id, event_type, url, secret, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetWebhookEndpointByID :one
SELECT id, app_id, event_type, url, secret, is_active, created_at, updated_at, deleted_at
FROM webhook_endpoints
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetWebhookEndpointByAppAndEvent :one
SELECT id, app_id, event_type, url, secret, is_active, created_at, updated_at, deleted_at
FROM webhook_endpoints
WHERE app_id = $1 AND event_type = $2 AND deleted_at IS NULL;

-- name: CountWebhookEndpointsByApp :one
SELECT COUNT(*) FROM webhook_endpoints
WHERE app_id = $1 AND deleted_at IS NULL;

-- name: ListWebhookEndpointsByApp :many
SELECT id, app_id, event_type, url, secret, is_active, created_at, updated_at, deleted_at
FROM webhook_endpoints
WHERE app_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAllWebhookEndpoints :one
SELECT COUNT(*) FROM webhook_endpoints
WHERE deleted_at IS NULL;

-- name: ListAllWebhookEndpoints :many
SELECT id, app_id, event_type, url, secret, is_active, created_at, updated_at, deleted_at
FROM webhook_endpoints
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetActiveWebhookEndpointsForEvent :many
SELECT id, app_id, event_type, url, secret, is_active, created_at, updated_at, deleted_at
FROM webhook_endpoints
WHERE app_id = $1 AND event_type = $2 AND is_active = true AND deleted_at IS NULL;

-- name: UpdateWebhookEndpointActive :exec
UPDATE webhook_endpoints
SET is_active = $2
WHERE id = $1;

-- name: SoftDeleteWebhookEndpoint :exec
UPDATE webhook_endpoints
SET deleted_at = NOW()
WHERE id = $1;

-- ============================================================================
-- Delivery operations
-- ============================================================================

-- name: CreateWebhookDelivery :exec
INSERT INTO webhook_deliveries (id, endpoint_id, app_id, event_type, payload, attempt, status_code, response_body, latency_ms, success, error_message, next_retry_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: CountWebhookDeliveriesByEndpoint :one
SELECT COUNT(*) FROM webhook_deliveries
WHERE endpoint_id = $1;

-- name: ListWebhookDeliveriesByEndpoint :many
SELECT id, endpoint_id, app_id, event_type, payload, attempt, status_code, response_body, latency_ms, success, error_message, next_retry_at, created_at
FROM webhook_deliveries
WHERE endpoint_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountWebhookDeliveriesByApp :one
SELECT COUNT(*) FROM webhook_deliveries
WHERE app_id = $1;

-- name: ListWebhookDeliveriesByApp :many
SELECT id, endpoint_id, app_id, event_type, payload, attempt, status_code, response_body, latency_ms, success, error_message, next_retry_at, created_at
FROM webhook_deliveries
WHERE app_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPendingWebhookRetries :many
SELECT id, endpoint_id, app_id, event_type, payload, attempt, status_code, response_body, latency_ms, success, error_message, next_retry_at, created_at
FROM webhook_deliveries
WHERE success = false AND next_retry_at IS NOT NULL AND next_retry_at <= $1
ORDER BY next_retry_at ASC
LIMIT $2;

-- name: ClearWebhookRetrySchedule :exec
UPDATE webhook_deliveries
SET next_retry_at = NULL
WHERE id = $1;
