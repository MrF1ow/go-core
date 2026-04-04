-- name: ClearUserLockout :exec
UPDATE users
SET locked_at = NULL,
    lock_reason = '',
    lock_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: LockUserByEmail :execrows
UPDATE users
SET locked_at = $1,
    lock_reason = $2,
    lock_expires_at = $3,
    updated_at = NOW()
WHERE app_id = $4 AND email = $5;
