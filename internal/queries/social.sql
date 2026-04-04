-- name: CreateSocialAccount :exec
INSERT INTO social_accounts (
    id, app_id, user_id, provider, provider_user_id,
    email, name, first_name, last_name, profile_picture,
    username, locale, raw_data, access_token, refresh_token,
    expires_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15,
    $16, $17, $18
);

-- name: GetOAuthProviderConfig :one
SELECT id, app_id, provider, client_id, client_secret, redirect_url, is_enabled, created_at, updated_at
FROM oauth_provider_configs
WHERE app_id = $1 AND provider = $2;

-- name: GetSocialAccountByProviderAndUserID :one
SELECT id, app_id, user_id, provider, provider_user_id,
       email, name, first_name, last_name, profile_picture,
       username, locale, raw_data, access_token, refresh_token,
       expires_at, created_at, updated_at
FROM social_accounts
WHERE app_id = $1 AND provider = $2 AND provider_user_id = $3;

-- name: GetSocialAccountsByUserID :many
SELECT id, app_id, user_id, provider, provider_user_id,
       email, name, first_name, last_name, profile_picture,
       username, locale, raw_data, access_token, refresh_token,
       expires_at, created_at, updated_at
FROM social_accounts
WHERE user_id = $1;

-- name: UpdateSocialAccount :exec
UPDATE social_accounts
SET app_id           = $2,
    user_id          = $3,
    provider         = $4,
    provider_user_id = $5,
    email            = $6,
    name             = $7,
    first_name       = $8,
    last_name        = $9,
    profile_picture  = $10,
    username         = $11,
    locale           = $12,
    raw_data         = $13,
    access_token     = $14,
    refresh_token    = $15,
    expires_at       = $16,
    updated_at       = NOW()
WHERE id = $1;

-- name: UpdateSocialAccountTokens :exec
UPDATE social_accounts
SET access_token  = $2,
    refresh_token = $3,
    updated_at    = NOW()
WHERE id = $1;

-- name: DeleteSocialAccount :exec
DELETE FROM social_accounts
WHERE id = $1;

-- name: GetSocialAccountByID :one
SELECT id, app_id, user_id, provider, provider_user_id,
       email, name, first_name, last_name, profile_picture,
       username, locale, raw_data, access_token, refresh_token,
       expires_at, created_at, updated_at
FROM social_accounts
WHERE id = $1;

-- name: CountSocialAccountsByUserID :one
SELECT COUNT(*) FROM social_accounts
WHERE user_id = $1;
