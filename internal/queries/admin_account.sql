-- name: CreateAdminAccount :one
INSERT INTO admin_accounts (username, email, password_hash, operator_role_id, app_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAdminAccountByUsername :one
SELECT * FROM admin_accounts
WHERE username = $1;

-- name: GetAdminAccountByUsernameOrEmail :one
SELECT * FROM admin_accounts
WHERE username = $1 OR email = $1;

-- name: GetAdminAccountByID :one
SELECT * FROM admin_accounts
WHERE id = $1;

-- name: GetAdminAccountByEmail :one
SELECT * FROM admin_accounts
WHERE email = $1;

-- name: UpdateAdminAccountLastLogin :exec
UPDATE admin_accounts
SET last_login_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListAllAdminAccounts :many
SELECT * FROM admin_accounts
ORDER BY created_at ASC;

-- name: CountAdminAccounts :one
SELECT COUNT(*) FROM admin_accounts;

-- name: DeleteAdminAccountByID :exec
DELETE FROM admin_accounts
WHERE id = $1;

-- name: UpdateAdminAccountEmail :exec
UPDATE admin_accounts
SET email = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdminAccountPassword :exec
UPDATE admin_accounts
SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: EnableAdminAccount2FA :exec
UPDATE admin_accounts
SET two_fa_enabled = TRUE,
    two_fa_method = $2,
    two_fa_secret = $3,
    two_fa_recovery_codes = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: DisableAdminAccount2FA :exec
UPDATE admin_accounts
SET two_fa_enabled = FALSE,
    two_fa_method = '',
    two_fa_secret = '',
    two_fa_recovery_codes = '[]',
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdminAccountRecoveryCodes :exec
UPDATE admin_accounts
SET two_fa_recovery_codes = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdminAccountMagicLinkEnabled :exec
UPDATE admin_accounts
SET magic_link_enabled = $2, updated_at = NOW()
WHERE id = $1;

-- name: SetAdminAccountBackupEmail :exec
UPDATE admin_accounts
SET backup_email = $2, backup_email_verified = FALSE, updated_at = NOW()
WHERE id = $1;

-- name: ClearAdminAccountBackupEmail :exec
UPDATE admin_accounts
SET backup_email = '', backup_email_verified = FALSE, updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdminAccountOperatorRole :exec
UPDATE admin_accounts
SET operator_role_id = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdminAccountAppID :exec
UPDATE admin_accounts
SET app_id = $2, updated_at = NOW()
WHERE id = $1;

-- name: SetAdminAccountDisabledAt :exec
UPDATE admin_accounts
SET disabled_at = $2, updated_at = NOW()
WHERE id = $1;

-- name: CountEnabledSuperadminAccounts :one
SELECT COUNT(*) FROM admin_accounts
WHERE operator_role_id = $1 AND disabled_at IS NULL AND app_id IS NULL;
