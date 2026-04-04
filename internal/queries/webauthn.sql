-- name: CreateWebAuthnCredential :exec
INSERT INTO webauthn_credentials (
    id, user_id, app_id, admin_id, credential_id, public_key,
    attestation_type, aaguid, sign_count, name, transports,
    backup_eligible, backup_state, last_used_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15
);

-- name: GetWebAuthnCredentialsByUserID :many
SELECT id, user_id, app_id, admin_id, credential_id, public_key,
       attestation_type, aaguid, sign_count, name, transports,
       backup_eligible, backup_state, last_used_at, created_at
FROM webauthn_credentials
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: GetWebAuthnCredentialsByUserAndApp :many
SELECT id, user_id, app_id, admin_id, credential_id, public_key,
       attestation_type, aaguid, sign_count, name, transports,
       backup_eligible, backup_state, last_used_at, created_at
FROM webauthn_credentials
WHERE user_id = $1 AND app_id = $2
ORDER BY created_at ASC;

-- name: GetWebAuthnCredentialByCredentialID :one
SELECT id, user_id, app_id, admin_id, credential_id, public_key,
       attestation_type, aaguid, sign_count, name, transports,
       backup_eligible, backup_state, last_used_at, created_at
FROM webauthn_credentials
WHERE credential_id = $1;

-- name: GetWebAuthnCredentialByAppAndCredentialID :one
SELECT id, user_id, app_id, admin_id, credential_id, public_key,
       attestation_type, aaguid, sign_count, name, transports,
       backup_eligible, backup_state, last_used_at, created_at
FROM webauthn_credentials
WHERE app_id = $1 AND credential_id = $2;

-- name: GetWebAuthnCredentialByID :one
SELECT id, user_id, app_id, admin_id, credential_id, public_key,
       attestation_type, aaguid, sign_count, name, transports,
       backup_eligible, backup_state, last_used_at, created_at
FROM webauthn_credentials
WHERE id = $1;

-- name: UpdateWebAuthnCredentialSignCount :exec
UPDATE webauthn_credentials
SET sign_count = $2, last_used_at = NOW()
WHERE id = $1;

-- name: DeleteWebAuthnCredential :exec
DELETE FROM webauthn_credentials
WHERE id = $1 AND user_id = $2;

-- name: CountWebAuthnCredentialsByUserAndApp :one
SELECT COUNT(*) FROM webauthn_credentials
WHERE user_id = $1 AND app_id = $2;

-- name: RenameWebAuthnCredential :execrows
UPDATE webauthn_credentials
SET name = $3
WHERE id = $1 AND user_id = $2;

-- name: GetWebAuthnCredentialsByAdminID :many
SELECT id, user_id, app_id, admin_id, credential_id, public_key,
       attestation_type, aaguid, sign_count, name, transports,
       backup_eligible, backup_state, last_used_at, created_at
FROM webauthn_credentials
WHERE admin_id = $1
ORDER BY created_at ASC;

-- name: DeleteWebAuthnAdminCredential :exec
DELETE FROM webauthn_credentials
WHERE id = $1 AND admin_id = $2;

-- name: RenameWebAuthnAdminCredential :execrows
UPDATE webauthn_credentials
SET name = $3
WHERE id = $1 AND admin_id = $2;

-- name: GetWebAuthnCredentialByAdminAndCredentialID :one
SELECT id, user_id, app_id, admin_id, credential_id, public_key,
       attestation_type, aaguid, sign_count, name, transports,
       backup_eligible, backup_state, last_used_at, created_at
FROM webauthn_credentials
WHERE admin_id = $1 AND credential_id = $2;
