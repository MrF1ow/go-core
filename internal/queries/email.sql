-- ============================================================================
-- Email Server Config queries
-- ============================================================================

-- name: GetServerConfigActiveDefault :one
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE app_id = $1 AND is_active = TRUE AND is_default = TRUE
LIMIT 1;

-- name: GetServerConfigActiveAny :one
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE app_id = $1 AND is_active = TRUE
LIMIT 1;

-- name: GetServerConfigByID :one
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE id = $1;

-- name: GetServerConfigAnyByApp :one
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE app_id = $1
LIMIT 1;

-- name: GetServerConfigsByApp :many
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE app_id = $1
ORDER BY is_default DESC, name ASC;

-- name: GetAllServerConfigs :many
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
ORDER BY app_id IS NOT NULL, app_id, is_default DESC, name ASC;

-- name: ListAllActiveServerConfigs :many
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE is_active = TRUE
ORDER BY is_default DESC, created_at ASC;

-- name: CreateServerConfig :exec
INSERT INTO email_server_configs (
    id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
    from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13, $14
);

-- name: UpdateServerConfig :exec
UPDATE email_server_configs
SET app_id        = $2,
    name          = $3,
    smtp_host     = $4,
    smtp_port     = $5,
    smtp_username = $6,
    smtp_password = $7,
    from_address  = $8,
    from_name     = $9,
    use_tls       = $10,
    is_active     = $11,
    is_default    = $12,
    updated_at    = $13
WHERE id = $1;

-- name: DeleteServerConfigByApp :exec
DELETE FROM email_server_configs WHERE app_id = $1;

-- name: DeleteServerConfigByID :exec
DELETE FROM email_server_configs WHERE id = $1;

-- name: ClearDefaultFlagGlobal :exec
UPDATE email_server_configs SET is_default = FALSE WHERE app_id IS NULL;

-- name: ClearDefaultFlagByApp :exec
UPDATE email_server_configs SET is_default = FALSE WHERE app_id = $1;

-- name: GetGlobalServerConfigActiveDefault :one
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE app_id IS NULL AND is_active = TRUE AND is_default = TRUE
LIMIT 1;

-- name: GetGlobalServerConfigActiveAny :one
SELECT id, app_id, name, smtp_host, smtp_port, smtp_username, smtp_password,
       from_address, from_name, use_tls, is_active, is_default, created_at, updated_at
FROM email_server_configs
WHERE app_id IS NULL AND is_active = TRUE
LIMIT 1;

-- ============================================================================
-- Email Type queries
-- ============================================================================

-- name: GetAllEmailTypes :many
SELECT id, code, name, description, default_subject, variables, is_system, is_active, created_at, updated_at
FROM email_types
ORDER BY name ASC;

-- name: GetActiveEmailTypes :many
SELECT id, code, name, description, default_subject, variables, is_system, is_active, created_at, updated_at
FROM email_types
WHERE is_active = TRUE
ORDER BY name ASC;

-- name: GetEmailTypeByCode :one
SELECT id, code, name, description, default_subject, variables, is_system, is_active, created_at, updated_at
FROM email_types
WHERE code = $1;

-- name: GetEmailTypeByID :one
SELECT id, code, name, description, default_subject, variables, is_system, is_active, created_at, updated_at
FROM email_types
WHERE id = $1;

-- name: CreateEmailType :exec
INSERT INTO email_types (
    id, code, name, description, default_subject, variables, is_system, is_active, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: UpdateEmailType :exec
UPDATE email_types
SET code            = $2,
    name            = $3,
    description     = $4,
    default_subject = $5,
    variables       = $6,
    is_system       = $7,
    is_active       = $8,
    updated_at      = $9
WHERE id = $1;

-- name: DeleteTemplatesByEmailTypeID :exec
DELETE FROM email_templates WHERE email_type_id = $1;

-- name: DeleteNonSystemEmailType :execrows
DELETE FROM email_types WHERE id = $1 AND is_system = FALSE;

-- ============================================================================
-- Email Template queries
-- ============================================================================

-- name: GetTemplateByAppAndType :one
SELECT id, app_id, email_type_id, server_config_id, name, subject,
       body_html, body_text, from_email, from_name, template_engine, is_active, created_at, updated_at
FROM email_templates
WHERE app_id = $1 AND email_type_id = $2 AND is_active = TRUE
LIMIT 1;

-- name: GetTemplateGlobalByType :one
SELECT id, app_id, email_type_id, server_config_id, name, subject,
       body_html, body_text, from_email, from_name, template_engine, is_active, created_at, updated_at
FROM email_templates
WHERE app_id IS NULL AND email_type_id = $1 AND is_active = TRUE
LIMIT 1;

-- name: GetTemplatesByApp :many
SELECT t.id, t.app_id, t.email_type_id, t.server_config_id, t.name, t.subject,
       t.body_html, t.body_text, t.from_email, t.from_name, t.template_engine, t.is_active,
       t.created_at, t.updated_at,
       et.id AS et_id, et.code AS et_code, et.name AS et_name, et.description AS et_description,
       et.default_subject AS et_default_subject, et.variables AS et_variables,
       et.is_system AS et_is_system, et.is_active AS et_is_active,
       et.created_at AS et_created_at, et.updated_at AS et_updated_at
FROM email_templates t
JOIN email_types et ON et.id = t.email_type_id
WHERE t.app_id = $1
ORDER BY t.created_at ASC;

-- name: GetGlobalDefaultTemplates :many
SELECT t.id, t.app_id, t.email_type_id, t.server_config_id, t.name, t.subject,
       t.body_html, t.body_text, t.from_email, t.from_name, t.template_engine, t.is_active,
       t.created_at, t.updated_at,
       et.id AS et_id, et.code AS et_code, et.name AS et_name, et.description AS et_description,
       et.default_subject AS et_default_subject, et.variables AS et_variables,
       et.is_system AS et_is_system, et.is_active AS et_is_active,
       et.created_at AS et_created_at, et.updated_at AS et_updated_at
FROM email_templates t
JOIN email_types et ON et.id = t.email_type_id
WHERE t.app_id IS NULL
ORDER BY t.created_at ASC;

-- name: GetTemplateByID :one
SELECT t.id, t.app_id, t.email_type_id, t.server_config_id, t.name, t.subject,
       t.body_html, t.body_text, t.from_email, t.from_name, t.template_engine, t.is_active,
       t.created_at, t.updated_at,
       et.id AS et_id, et.code AS et_code, et.name AS et_name, et.description AS et_description,
       et.default_subject AS et_default_subject, et.variables AS et_variables,
       et.is_system AS et_is_system, et.is_active AS et_is_active,
       et.created_at AS et_created_at, et.updated_at AS et_updated_at
FROM email_templates t
JOIN email_types et ON et.id = t.email_type_id
WHERE t.id = $1;

-- name: CreateTemplate :exec
INSERT INTO email_templates (
    id, app_id, email_type_id, server_config_id, name, subject,
    body_html, body_text, from_email, from_name, template_engine, is_active, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12, $13, $14
);

-- name: UpdateTemplate :exec
UPDATE email_templates
SET app_id           = $2,
    email_type_id    = $3,
    server_config_id = $4,
    name             = $5,
    subject          = $6,
    body_html        = $7,
    body_text        = $8,
    from_email       = $9,
    from_name        = $10,
    template_engine  = $11,
    is_active        = $12,
    updated_at       = $13
WHERE id = $1;

-- name: DeleteTemplateByID :exec
DELETE FROM email_templates WHERE id = $1;

-- name: GetTemplateByAppAndTypeID :one
SELECT id, app_id, email_type_id, server_config_id, name, subject,
       body_html, body_text, from_email, from_name, template_engine, is_active, created_at, updated_at
FROM email_templates
WHERE app_id = $1 AND email_type_id = $2
LIMIT 1;

-- name: GetTemplateGlobalByTypeID :one
SELECT id, app_id, email_type_id, server_config_id, name, subject,
       body_html, body_text, from_email, from_name, template_engine, is_active, created_at, updated_at
FROM email_templates
WHERE app_id IS NULL AND email_type_id = $1
LIMIT 1;
