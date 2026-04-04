-- name: GetApplicationForOIDC :one
SELECT id, tenant_id, name, description,
       two_fa_issuer_name, two_fa_enabled, two_fa_required, email_2fa_enabled, two_fa_methods,
       passkey_2fa_enabled, passkey_login_enabled,
       magic_link_enabled,
       login_notifications_enabled, suspicious_activity_alerts,
       sms_2fa_enabled,
       trusted_device_enabled, trusted_device_max_days,
       bf_lockout_enabled, bf_lockout_threshold, bf_lockout_durations, bf_lockout_window, bf_lockout_tier_ttl,
       bf_delay_enabled, bf_delay_start_after, bf_delay_max_seconds, bf_delay_tier_ttl,
       bf_captcha_enabled, bf_captcha_site_key, bf_captcha_secret_key, bf_captcha_threshold,
       oidc_enabled, oidc_rsa_private_key, oidc_id_token_ttl, oidc_issuer_url,
       frontend_url,
       login_logo_url, login_theme, login_primary_color, login_secondary_color, login_display_name,
       pw_min_length, pw_max_length, pw_require_upper, pw_require_lower, pw_require_digit, pw_require_symbol,
       pw_history_count, pw_max_age_days,
       access_token_ttl_minutes, refresh_token_ttl_hours,
       reset_password_path, magic_link_path, verify_email_path,
       created_at, updated_at
FROM applications
WHERE id = $1;

-- name: SaveOIDCRSAPrivateKey :exec
UPDATE applications
SET oidc_rsa_private_key = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: CreateOIDCClient :exec
INSERT INTO oidc_clients (
    id, app_id, name, description, client_id, client_secret_hash,
    redirect_uris, allowed_grant_types, allowed_scopes,
    require_consent, is_confidential, pkce_required, is_active,
    logo_url, login_theme, login_primary_color,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16,
    $17, $18
);

-- name: GetOIDCClientByID :one
SELECT id, app_id, name, description, client_id, client_secret_hash,
       redirect_uris, allowed_grant_types, allowed_scopes,
       require_consent, is_confidential, pkce_required, is_active,
       logo_url, login_theme, login_primary_color,
       created_at, updated_at
FROM oidc_clients
WHERE id = $1;

-- name: GetOIDCClientByClientID :one
SELECT id, app_id, name, description, client_id, client_secret_hash,
       redirect_uris, allowed_grant_types, allowed_scopes,
       require_consent, is_confidential, pkce_required, is_active,
       logo_url, login_theme, login_primary_color,
       created_at, updated_at
FROM oidc_clients
WHERE client_id = $1;

-- name: ListOIDCClientsByApp :many
SELECT id, app_id, name, description, client_id, client_secret_hash,
       redirect_uris, allowed_grant_types, allowed_scopes,
       require_consent, is_confidential, pkce_required, is_active,
       logo_url, login_theme, login_primary_color,
       created_at, updated_at
FROM oidc_clients
WHERE app_id = $1
ORDER BY created_at ASC;

-- name: UpdateOIDCClient :exec
UPDATE oidc_clients
SET app_id              = $2,
    name                = $3,
    description         = $4,
    client_id           = $5,
    client_secret_hash  = $6,
    redirect_uris       = $7,
    allowed_grant_types = $8,
    allowed_scopes      = $9,
    require_consent     = $10,
    is_confidential     = $11,
    pkce_required       = $12,
    is_active           = $13,
    logo_url            = $14,
    login_theme         = $15,
    login_primary_color = $16,
    updated_at          = NOW()
WHERE id = $1;

-- name: DeleteOIDCClient :exec
DELETE FROM oidc_clients
WHERE id = $1;

-- name: CreateOIDCAuthCode :exec
INSERT INTO oidc_auth_codes (
    id, app_id, client_id, user_id, code, redirect_uri,
    scopes, nonce, code_challenge, code_challenge_method,
    expires_at, used, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10,
    $11, $12, $13
);

-- name: GetOIDCAuthCode :one
SELECT id, app_id, client_id, user_id, code, redirect_uri,
       scopes, nonce, code_challenge, code_challenge_method,
       expires_at, used, created_at
FROM oidc_auth_codes
WHERE code = $1;

-- name: MarkOIDCAuthCodeUsed :exec
UPDATE oidc_auth_codes
SET used = TRUE
WHERE id = $1;

-- name: DeleteExpiredOIDCAuthCodes :exec
DELETE FROM oidc_auth_codes
WHERE expires_at < NOW();

-- name: GetUserByIDForOIDC :one
SELECT id, app_id, email, password_hash, email_verified, is_active,
       name, first_name, last_name, profile_picture, locale,
       two_fa_enabled, two_fa_method, two_fa_secret, two_fa_recovery_codes,
       backup_email, backup_email_verified,
       two_fa_previous_method, two_fa_previous_secret,
       phone_number, phone_verified,
       locked_at, lock_reason, lock_expires_at,
       password_history, password_changed_at,
       created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmailForOIDC :one
SELECT id, app_id, email, password_hash, email_verified, is_active,
       name, first_name, last_name, profile_picture, locale,
       two_fa_enabled, two_fa_method, two_fa_secret, two_fa_recovery_codes,
       backup_email, backup_email_verified,
       two_fa_previous_method, two_fa_previous_secret,
       phone_number, phone_verified,
       locked_at, lock_reason, lock_expires_at,
       password_history, password_changed_at,
       created_at, updated_at
FROM users
WHERE app_id = $1 AND email = $2;
