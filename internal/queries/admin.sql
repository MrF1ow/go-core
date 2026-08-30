-- ============ TENANTS ============

-- name: AdminCreateTenant :one
INSERT INTO tenants (id, name, created_at, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: AdminGetTenantByID :one
SELECT * FROM tenants WHERE id = $1;

-- name: AdminGetTenantApps :many
SELECT * FROM applications WHERE tenant_id = $1;

-- name: AdminCountTenants :one
SELECT COUNT(*) FROM tenants;

-- name: AdminListTenants :many
SELECT * FROM tenants
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminListTenantsWithAppCount :many
SELECT t.id, t.name, t.created_at, t.updated_at,
       COUNT(a.id) AS app_count
FROM tenants t
LEFT JOIN applications a ON a.tenant_id = t.id
GROUP BY t.id
ORDER BY t.created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminUpdateTenant :exec
UPDATE tenants SET name = $2, updated_at = NOW() WHERE id = $1;

-- name: AdminDeleteTenant :exec
DELETE FROM tenants WHERE id = $1;

-- name: AdminListAllTenants :many
SELECT id, name FROM tenants ORDER BY name ASC;

-- ============ APPLICATIONS ============

-- name: AdminCreateApp :one
INSERT INTO applications (
    id, tenant_id, name, description,
    two_fa_issuer_name, two_fa_enabled, two_fa_required,
    email_2fa_enabled, two_fa_methods,
    passkey_2fa_enabled, passkey_login_enabled,
    magic_link_enabled,
    login_notifications_enabled, suspicious_activity_alerts,
    sms_2fa_enabled,
    trusted_device_enabled, trusted_device_max_days,
    oidc_enabled, oidc_rsa_private_key, oidc_id_token_ttl, oidc_issuer_url,
    frontend_url,
    login_logo_url, login_theme, login_primary_color, login_secondary_color, login_display_name,
    pw_min_length, pw_max_length, pw_require_upper, pw_require_lower,
    pw_require_digit, pw_require_symbol, pw_history_count, pw_max_age_days,
    access_token_ttl_minutes, refresh_token_ttl_hours,
    reset_password_path, magic_link_path, verify_email_path,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7,
    $8, $9,
    $10, $11,
    $12,
    $13, $14,
    $15,
    $16, $17,
    $18, $19, $20, $21,
    $22,
    $23, $24, $25, $26, $27,
    $28, $29, $30, $31,
    $32, $33, $34, $35,
    $36, $37,
    $38, $39, $40,
    $41, $42
)
RETURNING *;

-- name: AdminGetAppByID :one
SELECT * FROM applications WHERE id = $1;

-- name: AdminGetAppOAuthConfigs :many
SELECT * FROM oauth_provider_configs WHERE app_id = $1;

-- name: AdminListAppsByTenant :many
SELECT * FROM applications WHERE tenant_id = $1;

-- name: AdminCountApps :one
SELECT COUNT(*) FROM applications;

-- name: AdminCountAppsWithTenantFilter :one
SELECT COUNT(*) FROM applications
WHERE (sqlc.narg('tenant_id')::uuid IS NULL OR tenant_id = sqlc.narg('tenant_id')::uuid);

-- name: AdminListAppsWithDetails :many
SELECT a.id, a.tenant_id, a.name, a.description,
       a.created_at, a.updated_at,
       a.two_fa_enabled, a.two_fa_required,
       a.passkey_2fa_enabled, a.passkey_login_enabled,
       a.magic_link_enabled,
       t.name AS tenant_name,
       COUNT(o.id) AS oauth_config_count
FROM applications a
LEFT JOIN tenants t ON t.id = a.tenant_id
LEFT JOIN oauth_provider_configs o ON o.app_id = a.id
WHERE (sqlc.narg('tenant_id')::uuid IS NULL OR a.tenant_id = sqlc.narg('tenant_id')::uuid)
GROUP BY a.id, t.name
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminUpdateApp :exec
UPDATE applications SET
    name = $2,
    description = $3,
    frontend_url = $4,
    two_fa_issuer_name = $5,
    two_fa_enabled = $6,
    two_fa_required = $7,
    passkey_2fa_enabled = $8,
    passkey_login_enabled = $9,
    magic_link_enabled = $10,
    oidc_enabled = $11,
    bf_lockout_enabled = $12,
    bf_lockout_threshold = $13,
    bf_lockout_durations = $14,
    bf_lockout_window = $15,
    bf_lockout_tier_ttl = $16,
    bf_delay_enabled = $17,
    bf_delay_start_after = $18,
    bf_delay_max_seconds = $19,
    bf_delay_tier_ttl = $20,
    bf_captcha_enabled = $21,
    bf_captcha_site_key = $22,
    bf_captcha_threshold = $23,
    login_logo_url = $24,
    login_primary_color = $25,
    login_secondary_color = $26,
    login_display_name = $27,
    pw_min_length = $28,
    pw_max_length = $29,
    pw_require_upper = $30,
    pw_require_lower = $31,
    pw_require_digit = $32,
    pw_require_symbol = $33,
    pw_history_count = $34,
    pw_max_age_days = $35,
    access_token_ttl_minutes = $36,
    refresh_token_ttl_hours = $37,
    reset_password_path = $38,
    magic_link_path = $39,
    verify_email_path = $40,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminUpdateAppCaptchaSecretKey :exec
UPDATE applications SET bf_captcha_secret_key = $2, updated_at = NOW() WHERE id = $1;

-- name: AdminDeleteApp :exec
DELETE FROM applications WHERE id = $1;

-- name: AdminUpdateAppSMSTrustedDevice :exec
UPDATE applications SET
    sms_2fa_enabled = $2,
    trusted_device_enabled = $3,
    trusted_device_max_days = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminListAllAppsWithTenantName :many
SELECT a.id, a.name, t.name AS tenant_name
FROM applications a
LEFT JOIN tenants t ON t.id = a.tenant_id
ORDER BY t.name ASC, a.name ASC;

-- name: AdminListAllAppIDs :many
SELECT id::text FROM applications;

-- ============ OAUTH CONFIGS ============

-- name: AdminGetOAuthConfigByAppAndProvider :one
SELECT * FROM oauth_provider_configs
WHERE app_id = $1 AND provider = $2;

-- name: AdminCreateOAuthConfig :one
INSERT INTO oauth_provider_configs (id, app_id, provider, client_id, client_secret, redirect_url, is_enabled, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: AdminUpdateOAuthConfig :exec
UPDATE oauth_provider_configs SET
    client_id = $2,
    client_secret = $3,
    redirect_url = $4,
    is_enabled = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminCountOAuthConfigs :one
SELECT COUNT(*) FROM oauth_provider_configs
WHERE (sqlc.narg('app_id')::uuid IS NULL OR app_id = sqlc.narg('app_id')::uuid);

-- name: AdminListOAuthConfigsWithDetails :many
SELECT o.id, o.app_id, o.provider, o.client_id, o.redirect_url, o.is_enabled,
       o.created_at, o.updated_at,
       a.name AS app_name,
       t.name AS tenant_name
FROM oauth_provider_configs o
LEFT JOIN applications a ON a.id = o.app_id
LEFT JOIN tenants t ON t.id = a.tenant_id
WHERE (sqlc.narg('app_id')::uuid IS NULL OR o.app_id = sqlc.narg('app_id')::uuid)
ORDER BY o.created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminGetOAuthConfigByID :one
SELECT * FROM oauth_provider_configs WHERE id = $1;

-- name: AdminUpdateOAuthConfigByID :exec
UPDATE oauth_provider_configs SET
    client_id = $2,
    redirect_url = $3,
    is_enabled = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminUpdateOAuthConfigByIDWithSecret :exec
UPDATE oauth_provider_configs SET
    client_id = $2,
    client_secret = $3,
    redirect_url = $4,
    is_enabled = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminDeleteOAuthConfig :exec
DELETE FROM oauth_provider_configs WHERE id = $1;

-- name: AdminToggleOAuthConfigEnabled :one
UPDATE oauth_provider_configs SET
    is_enabled = NOT is_enabled,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AdminGetEnabledOAuthProviders :many
SELECT provider FROM oauth_provider_configs
WHERE app_id = $1 AND is_enabled = true;

-- ============ OIDC CLIENTS ============

-- name: AdminCountActiveOIDCClients :one
SELECT COUNT(*) FROM oidc_clients WHERE app_id = $1 AND is_active = true;

-- name: AdminGetFirstActiveOIDCClientLoginTheme :one
SELECT login_theme FROM oidc_clients
WHERE app_id = $1 AND is_active = true
ORDER BY created_at ASC
LIMIT 1;

-- ============ USERS ============

-- name: AdminCountUsersWithFilters :one
SELECT COUNT(*)
FROM users u
LEFT JOIN applications a ON a.id = u.app_id
LEFT JOIN tenants t ON t.id = a.tenant_id
WHERE (sqlc.narg('app_id')::uuid IS NULL OR u.app_id = sqlc.narg('app_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR (u.email ILIKE '%' || sqlc.narg('search')::text || '%' OR u.name ILIKE '%' || sqlc.narg('search')::text || '%'));

-- name: AdminListUsersWithDetails :many
SELECT u.id, u.email, u.name, u.app_id,
       a.name AS app_name,
       COALESCE(t.name, '') AS tenant_name,
       u.is_active, u.email_verified, u.two_fa_enabled,
       (u.password_hash != '') AS has_password,
       COALESCE(sa_count.count, 0)::bigint AS social_account_count,
       u.locked_at, u.lock_expires_at,
       u.created_at
FROM users u
LEFT JOIN applications a ON a.id = u.app_id
LEFT JOIN tenants t ON t.id = a.tenant_id
LEFT JOIN (SELECT user_id, COUNT(*) AS count FROM social_accounts GROUP BY user_id) sa_count ON sa_count.user_id = u.id
WHERE (sqlc.narg('app_id')::uuid IS NULL OR u.app_id = sqlc.narg('app_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR (u.email ILIKE '%' || sqlc.narg('search')::text || '%' OR u.name ILIKE '%' || sqlc.narg('search')::text || '%'))
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminGetUserDetailByID :one
SELECT u.id, u.email, u.name, u.first_name, u.last_name,
       u.profile_picture, u.locale, u.app_id,
       a.name AS app_name,
       COALESCE(t.name, '') AS tenant_name,
       u.is_active, u.email_verified, u.two_fa_enabled,
       (u.password_hash != '') AS has_password,
       COALESCE(u.backup_email, '') AS backup_email,
       u.backup_email_verified,
       COALESCE(u.phone_number, '') AS phone_number,
       u.phone_verified,
       u.locked_at, u.lock_reason, u.lock_expires_at,
       u.created_at, u.updated_at
FROM users u
LEFT JOIN applications a ON a.id = u.app_id
LEFT JOIN tenants t ON t.id = a.tenant_id
WHERE u.id = $1;

-- name: AdminGetUserActiveAndAppID :one
SELECT id, is_active, app_id FROM users WHERE id = $1;

-- name: AdminSetUserActive :exec
UPDATE users SET is_active = $2, updated_at = NOW() WHERE id = $1;

-- name: AdminGetUserEmailAndAppID :one
SELECT id, email, app_id FROM users WHERE id = $1;

-- name: AdminUnlockUser :exec
UPDATE users SET locked_at = NULL, lock_reason = '', lock_expires_at = NULL, updated_at = NOW()
WHERE id = $1;

-- name: AdminCountActiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = true;

-- name: AdminCountInactiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = false;

-- name: AdminGetUserIDByEmailAndApp :one
SELECT id FROM users WHERE app_id = $1 AND email = $2 LIMIT 1;

-- name: AdminGetUserEmailsByIDs :many
SELECT id, email FROM users WHERE id = ANY($1::uuid[]);

-- name: AdminGetAppNamesByIDs :many
SELECT id, name FROM applications WHERE id = ANY($1::uuid[]);

-- name: AdminCountUserExistsForImport :one
SELECT COUNT(*) FROM users WHERE email = $1 AND app_id = $2;

-- name: AdminCreateImportUser :one
INSERT INTO users (id, app_id, email, name, first_name, last_name, locale, password_hash, email_verified, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, '', false, true, NOW(), NOW())
RETURNING *;

-- name: AdminGetUsersByEmailsAndApp :many
SELECT id, email FROM users WHERE email = ANY($1::text[]) AND app_id = $2;

-- ============ USER EXPORT ============

-- name: AdminExportUsers :many
SELECT u.id, u.app_id, u.email, u.name,
       u.first_name, u.last_name, u.locale,
       u.email_verified, u.is_active,
       u.two_fa_enabled, u.two_fa_method,
       COALESCE(sa.providers, '') AS social_providers,
       u.created_at, u.updated_at
FROM users u
LEFT JOIN (SELECT user_id, STRING_AGG(provider, ',') AS providers FROM social_accounts GROUP BY user_id) sa ON sa.user_id = u.id
WHERE (sqlc.narg('app_id')::uuid IS NULL OR u.app_id = sqlc.narg('app_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR (u.email ILIKE '%' || sqlc.narg('search')::text || '%' OR u.name ILIKE '%' || sqlc.narg('search')::text || '%'))
ORDER BY u.created_at DESC
LIMIT $1;

-- ============ ACTIVITY LOGS ============

-- name: AdminCountActivityLogs :one
SELECT COUNT(*)
FROM activity_logs al
LEFT JOIN users u ON u.id = al.user_id
LEFT JOIN applications a ON a.id = al.app_id
WHERE (sqlc.narg('event_type')::text IS NULL OR al.event_type = sqlc.narg('event_type')::text)
  AND (sqlc.narg('severity')::text IS NULL OR al.severity = sqlc.narg('severity')::text)
  AND (sqlc.narg('app_id')::uuid IS NULL OR al.app_id = sqlc.narg('app_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR u.email ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR al.timestamp >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR al.timestamp <= sqlc.narg('end_date')::timestamptz);

-- name: AdminListActivityLogs :many
SELECT al.id, al.app_id,
       COALESCE(a.name, '') AS app_name,
       al.user_id,
       COALESCE(u.email, '') AS user_email,
       al.event_type, al.severity,
       al.ip_address, al.is_anomaly,
       al.timestamp
FROM activity_logs al
LEFT JOIN users u ON u.id = al.user_id
LEFT JOIN applications a ON a.id = al.app_id
WHERE (sqlc.narg('event_type')::text IS NULL OR al.event_type = sqlc.narg('event_type')::text)
  AND (sqlc.narg('severity')::text IS NULL OR al.severity = sqlc.narg('severity')::text)
  AND (sqlc.narg('app_id')::uuid IS NULL OR al.app_id = sqlc.narg('app_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR u.email ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR al.timestamp >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR al.timestamp <= sqlc.narg('end_date')::timestamptz)
ORDER BY al.timestamp DESC
LIMIT $1 OFFSET $2;

-- name: AdminGetActivityLogDetail :one
SELECT al.id, al.app_id,
       COALESCE(a.name, '') AS app_name,
       al.user_id,
       COALESCE(u.email, '') AS user_email,
       al.event_type, al.severity,
       al.ip_address, al.user_agent,
       COALESCE(al.details::text, '') AS details,
       al.is_anomaly, al.expires_at,
       al.timestamp
FROM activity_logs al
LEFT JOIN users u ON u.id = al.user_id
LEFT JOIN applications a ON a.id = al.app_id
WHERE al.id = $1;

-- name: AdminListDistinctEventTypes :many
SELECT DISTINCT event_type FROM activity_logs ORDER BY event_type ASC;

-- name: AdminListDistinctSeverities :many
SELECT DISTINCT severity FROM activity_logs ORDER BY severity ASC;

-- name: AdminExportActivityLogs :many
SELECT al.id, al.app_id,
       COALESCE(a.name, '') AS app_name,
       al.user_id,
       COALESCE(u.email, '') AS user_email,
       al.event_type, al.severity,
       al.ip_address, al.user_agent,
       al.is_anomaly,
       al.timestamp
FROM activity_logs al
LEFT JOIN users u ON u.id = al.user_id
LEFT JOIN applications a ON a.id = al.app_id
WHERE (sqlc.narg('event_type')::text IS NULL OR al.event_type = sqlc.narg('event_type')::text)
  AND (sqlc.narg('severity')::text IS NULL OR al.severity = sqlc.narg('severity')::text)
  AND (sqlc.narg('app_id')::uuid IS NULL OR al.app_id = sqlc.narg('app_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR u.email ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (sqlc.narg('start_date')::timestamptz IS NULL OR al.timestamp >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz IS NULL OR al.timestamp <= sqlc.narg('end_date')::timestamptz)
ORDER BY al.timestamp DESC
LIMIT $1;

-- ============ API KEYS ============

-- name: AdminCreateApiKey :one
INSERT INTO api_keys (id, key_type, name, description, key_hash, key_prefix, key_suffix, app_id, scopes, operator_role_id, expires_at, is_revoked, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: AdminCountApiKeys :one
SELECT COUNT(*)
FROM api_keys
WHERE (sqlc.narg('key_type')::text IS NULL OR key_type = sqlc.narg('key_type')::text)
  AND (sqlc.narg('app_id')::uuid IS NULL OR app_id = sqlc.narg('app_id')::uuid);

-- name: AdminListApiKeys :many
SELECT ak.id, ak.key_type, ak.name,
       ak.key_prefix, ak.key_suffix, ak.scopes,
       ak.app_id, ak.expires_at,
       ak.last_used_at, ak.is_revoked,
       ak.created_at,
       COALESCE(a.name, '') AS app_name,
       COALESCE(t.name, '') AS tenant_name,
       COALESCE(r.name, '') AS operator_role_name
FROM api_keys ak
LEFT JOIN applications a ON a.id = ak.app_id
LEFT JOIN tenants t ON t.id = a.tenant_id
LEFT JOIN operator_roles r ON r.id = ak.operator_role_id
WHERE (sqlc.narg('key_type')::text IS NULL OR ak.key_type = sqlc.narg('key_type')::text)
  AND (sqlc.narg('app_id')::uuid IS NULL OR ak.app_id = sqlc.narg('app_id')::uuid)
ORDER BY ak.created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminGetApiKeyByID :one
SELECT * FROM api_keys WHERE id = $1;

-- name: AdminRevokeApiKey :exec
UPDATE api_keys SET is_revoked = true, updated_at = NOW() WHERE id = $1;

-- name: AdminFindActiveKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = $1 AND is_revoked = false;

-- name: AdminUpdateApiKeyLastUsed :exec
UPDATE api_keys SET last_used_at = NOW() WHERE id = $1;

-- name: AdminUpdateApiKey :exec
UPDATE api_keys SET name = $2, description = $3, scopes = $4, operator_role_id = $5, expires_at = $6, updated_at = NOW()
WHERE id = $1;

-- name: AdminGetKeysExpiringWithin :many
SELECT * FROM api_keys
WHERE is_revoked = false
  AND expires_at IS NOT NULL
  AND expires_at > NOW()
  AND expires_at <= $1;

-- name: AdminMarkApiKeyNotified7Days :exec
UPDATE api_keys SET notified_7_days_at = NOW() WHERE id = $1;

-- name: AdminMarkApiKeyNotified1Day :exec
UPDATE api_keys SET notified_1_day_at = NOW() WHERE id = $1;

-- name: AdminIncrementDailyUsage :exec
INSERT INTO api_key_usages (api_key_id, period_date, request_count, updated_at)
VALUES ($1, $2, 1, NOW())
ON CONFLICT (api_key_id, period_date)
DO UPDATE SET request_count = api_key_usages.request_count + 1, updated_at = NOW();

-- name: AdminGetApiKeyUsageSummary :many
SELECT period_date, request_count
FROM api_key_usages
WHERE api_key_id = $1 AND period_date >= $2
ORDER BY period_date ASC;

-- name: AdminGetApiKeyTotalUsage :one
SELECT COALESCE(SUM(request_count), 0)::bigint AS total
FROM api_key_usages
WHERE api_key_id = $1;

-- ============ SOCIAL ACCOUNTS ============

-- name: AdminGetSocialAccountByID :one
SELECT * FROM social_accounts WHERE id = $1;

-- name: AdminDeleteSocialAccount :exec
DELETE FROM social_accounts WHERE id = $1;

-- name: AdminCountSocialAccountsByUserID :one
SELECT COUNT(*) FROM social_accounts WHERE user_id = $1;

-- name: AdminGetSocialAccountsByUserID :many
SELECT * FROM social_accounts WHERE user_id = $1 ORDER BY created_at ASC;

-- ============ WEBAUTHN CREDENTIALS ============

-- name: AdminGetWebAuthnCredentialByID :one
SELECT * FROM webauthn_credentials WHERE id = $1;

-- name: AdminDeleteWebAuthnCredential :exec
DELETE FROM webauthn_credentials WHERE id = $1;

-- name: AdminGetWebAuthnCredentialsByUserID :many
SELECT * FROM webauthn_credentials WHERE user_id = $1 ORDER BY created_at ASC;

-- ============ PERMISSIONS & ROLES (for SeedDefaultRolesForApp) ============

-- name: AdminListAllPermissions :many
SELECT * FROM permissions;

-- name: AdminGetRoleByAppAndName :one
SELECT * FROM roles WHERE app_id = $1 AND name = $2;

-- name: AdminCreateRole :one
INSERT INTO roles (id, app_id, name, description, is_system, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING *;

-- name: AdminDeleteRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: AdminAddRolePermission :exec
INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- ============ SESSION GROUPS ============

-- name: AdminCreateSessionGroup :one
INSERT INTO session_groups (id, tenant_id, name, description, global_logout, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: AdminGetSessionGroupByID :one
SELECT * FROM session_groups WHERE id = $1;

-- name: AdminCountSessionGroups :one
SELECT COUNT(*) FROM session_groups;

-- name: AdminListSessionGroups :many
SELECT sg.id, sg.tenant_id, t.name AS tenant_name, sg.name, sg.description,
       sg.global_logout, sg.created_at,
       COUNT(sga.app_id) AS app_count
FROM session_groups sg
LEFT JOIN tenants t ON t.id = sg.tenant_id
LEFT JOIN session_group_apps sga ON sga.session_group_id = sg.id
GROUP BY sg.id, t.name
ORDER BY sg.created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminUpdateSessionGroup :exec
UPDATE session_groups SET name = $2, description = $3, global_logout = $4, updated_at = NOW()
WHERE id = $1;

-- name: AdminDeleteSessionGroupApps :exec
DELETE FROM session_group_apps WHERE session_group_id = $1;

-- name: AdminDeleteSessionGroup :exec
DELETE FROM session_groups WHERE id = $1;

-- name: AdminAddAppToSessionGroup :exec
INSERT INTO session_group_apps (session_group_id, app_id, added_at) VALUES ($1, $2, NOW());

-- name: AdminRemoveAppFromSessionGroup :exec
DELETE FROM session_group_apps WHERE session_group_id = $1 AND app_id = $2;

-- name: AdminGetSessionGroupForApp :one
SELECT sg.*
FROM session_group_apps sga
JOIN session_groups sg ON sg.id = sga.session_group_id
WHERE sga.app_id = $1;

-- name: AdminGetAppsInSessionGroup :many
SELECT app_id FROM session_group_apps WHERE session_group_id = $1;

-- name: AdminGetPeersForApp :many
SELECT sga.app_id::text, a.frontend_url
FROM session_group_apps sga
JOIN applications a ON a.id = sga.app_id
WHERE sga.session_group_id = (SELECT sga2.session_group_id FROM session_group_apps sga2 WHERE sga2.app_id = $1)
  AND sga.app_id != $1;

-- name: AdminGetAppsInSessionGroupWithDetails :many
SELECT sga.app_id, a.name AS app_name, t.name AS tenant_name
FROM session_group_apps sga
JOIN applications a ON a.id = sga.app_id
LEFT JOIN tenants t ON t.id = a.tenant_id
WHERE sga.session_group_id = $1
ORDER BY t.name ASC, a.name ASC;

-- ============ DASHBOARD ============

-- name: AdminCountTotalUsers :one
SELECT COUNT(*) FROM users;

-- name: AdminCountRecentActivityLogs :one
SELECT COUNT(*) FROM activity_logs WHERE timestamp >= $1;

-- name: AdminCountActiveTrustedDevices :one
SELECT COUNT(*) FROM trusted_devices WHERE expires_at > NOW();

-- name: AdminCountVerifiedPhoneUsers :one
SELECT COUNT(*) FROM users WHERE phone_verified = true;

-- name: AdminGetRecentActivity :many
SELECT * FROM activity_logs ORDER BY timestamp DESC LIMIT $1;

-- name: AdminCountTotalUsersByApp :one
SELECT COUNT(*) FROM users WHERE app_id = $1;

-- name: AdminCountActiveUsersByApp :one
SELECT COUNT(*) FROM users WHERE is_active = true AND app_id = $1;

-- name: AdminCountRecentActivityLogsByApp :one
SELECT COUNT(*) FROM activity_logs WHERE timestamp >= $1 AND app_id = $2;

-- name: AdminCountActiveTrustedDevicesByApp :one
SELECT COUNT(*) FROM trusted_devices WHERE expires_at > NOW() AND app_id = $1;

-- name: AdminCountVerifiedPhoneUsersByApp :one
SELECT COUNT(*) FROM users WHERE phone_verified = true AND app_id = $1;

-- name: AdminGetRecentActivityByApp :many
SELECT * FROM activity_logs WHERE app_id = $1 ORDER BY timestamp DESC LIMIT $2;
