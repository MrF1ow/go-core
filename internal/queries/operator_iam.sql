-- name: InsertOperatorIAMEvent :one
INSERT INTO operator_iam_events (
    actor_kind, actor_key_id, actor_account_id,
    target_kind, target_key_id, target_account_id,
    old_role_id, new_role_id, action
) VALUES (
    $1, $2, $3,
    $4, $5, $6,
    $7, $8, $9
)
RETURNING *;

-- name: ListOperatorIAMEvents :many
SELECT *
FROM operator_iam_events
WHERE (sqlc.narg('target_key_id')::uuid IS NULL OR target_key_id = sqlc.narg('target_key_id')::uuid)
  AND (sqlc.narg('target_account_id')::uuid IS NULL OR target_account_id = sqlc.narg('target_account_id')::uuid)
ORDER BY at DESC
LIMIT $1 OFFSET $2;

-- name: InsertOperatorAccessLog :one
INSERT INTO operator_access_logs (
    kind, key_id, account_id, role_name, method, path, decision, resource, action, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: ListOperatorAccessLogs :many
SELECT *
FROM operator_access_logs
WHERE (sqlc.narg('decision')::text IS NULL OR decision = sqlc.narg('decision')::text)
ORDER BY at DESC
LIMIT $1 OFFSET $2;
