-- name: CreateUser :one
INSERT INTO users (
    id, app_id, email, password_hash, email_verified, is_active,
    name, first_name, last_name, profile_picture, locale,
    two_fa_enabled, two_fa_method, two_fa_secret, two_fa_recovery_codes,
    backup_email, backup_email_verified,
    two_fa_previous_method, two_fa_previous_secret,
    phone_number, phone_verified,
    locked_at, lock_reason, lock_expires_at,
    password_history, password_changed_at,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15,
    $16, $17,
    $18, $19,
    $20, $21,
    $22, $23, $24,
    $25, $26,
    $27, $28
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE app_id = $1 AND email = $2;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateUser :exec
UPDATE users
SET app_id               = $2,
    email                = $3,
    password_hash        = $4,
    email_verified       = $5,
    is_active            = $6,
    name                 = $7,
    first_name           = $8,
    last_name            = $9,
    profile_picture      = $10,
    locale               = $11,
    two_fa_enabled       = $12,
    two_fa_method        = $13,
    two_fa_secret        = $14,
    two_fa_recovery_codes = $15,
    backup_email         = $16,
    backup_email_verified = $17,
    two_fa_previous_method = $18,
    two_fa_previous_secret = $19,
    phone_number         = $20,
    phone_verified       = $21,
    locked_at            = $22,
    lock_reason          = $23,
    lock_expires_at      = $24,
    password_history     = $25,
    password_changed_at  = $26,
    updated_at           = NOW()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserPasswordWithHistory :exec
UPDATE users
SET password_hash       = $2,
    password_history    = $3,
    password_changed_at = $4,
    updated_at          = NOW()
WHERE id = $1;

-- name: UpdateUserEmailVerified :exec
UPDATE users
SET email_verified = $2, updated_at = NOW()
WHERE id = $1;

-- name: Enable2FAWithMethod :exec
UPDATE users
SET two_fa_enabled        = TRUE,
    two_fa_secret         = $2,
    two_fa_recovery_codes = $3,
    two_fa_method         = $4,
    updated_at            = NOW()
WHERE id = $1;

-- name: Disable2FA :exec
UPDATE users
SET two_fa_enabled        = FALSE,
    two_fa_secret         = '',
    two_fa_recovery_codes = NULL,
    two_fa_method         = '',
    updated_at            = NOW()
WHERE id = $1;

-- name: UpdateRecoveryCodes :exec
UPDATE users
SET two_fa_recovery_codes = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserProfile :exec
UPDATE users
SET name            = $2,
    first_name      = $3,
    last_name       = $4,
    profile_picture = $5,
    locale          = $6,
    updated_at      = NOW()
WHERE id = $1;

-- name: UpdateUserEmail :exec
UPDATE users
SET email          = $2,
    email_verified = FALSE,
    updated_at     = NOW()
WHERE id = $1;

-- name: ClearLockout :exec
UPDATE users
SET locked_at       = NULL,
    lock_reason     = '',
    lock_expires_at = NULL,
    updated_at      = NOW()
WHERE id = $1;

-- name: SetBackupEmail :exec
UPDATE users
SET backup_email          = $2,
    backup_email_verified = FALSE,
    updated_at            = NOW()
WHERE id = $1;

-- name: VerifyBackupEmail :exec
UPDATE users
SET backup_email_verified = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: ClearBackupEmail :exec
UPDATE users
SET backup_email          = '',
    backup_email_verified = FALSE,
    updated_at            = NOW()
WHERE id = $1;

-- name: SaveAndSwitchToBackupEmail2FA :exec
UPDATE users
SET two_fa_previous_method = $2,
    two_fa_previous_secret = $3,
    two_fa_enabled         = TRUE,
    two_fa_method          = 'backup_email',
    two_fa_secret          = '',
    two_fa_recovery_codes  = $4,
    updated_at             = NOW()
WHERE id = $1;

-- name: GetUserTwoFAPreviousFields :one
SELECT two_fa_previous_method, two_fa_previous_secret
FROM users
WHERE id = $1;

-- name: RestorePreviousTwoFAMethod :exec
UPDATE users
SET two_fa_method          = $2,
    two_fa_secret          = $3,
    two_fa_enabled         = $4,
    two_fa_previous_method = '',
    two_fa_previous_secret = '',
    updated_at             = NOW()
WHERE id = $1;

-- name: SetPhoneNumber :exec
UPDATE users
SET phone_number   = $2,
    phone_verified = FALSE,
    updated_at     = NOW()
WHERE id = $1;

-- name: VerifyPhoneNumber :exec
UPDATE users
SET phone_verified = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: ClearPhone :exec
UPDATE users
SET phone_number   = '',
    phone_verified = FALSE,
    updated_at     = NOW()
WHERE id = $1;

-- name: DeleteUserByID :exec
DELETE FROM users
WHERE id = $1;
