-- Consolidated schema for SQLC code generation.
-- This file represents the final state of the database after all migrations.
-- It is NOT run against the database — the incremental migrations/ files are used for that.
-- Update this file when adding new migrations.

-- ─── schema_migrations ───────────────────────────────────────────────────────

CREATE TABLE schema_migrations (
    id                SERIAL PRIMARY KEY,
    version           VARCHAR(255) NOT NULL UNIQUE,
    name              VARCHAR(255) NOT NULL,
    applied_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    execution_time_ms INTEGER,
    success           BOOLEAN      NOT NULL DEFAULT TRUE,
    error_message     TEXT,
    checksum          VARCHAR(64)
);

-- ─── tenants ─────────────────────────────────────────────────────────────────

CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── applications ─────────────────────────────────────────────────────────────

CREATE TABLE applications (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                     TEXT         NOT NULL,
    description              TEXT,
    -- 2FA settings
    two_fa_issuer_name       TEXT         NOT NULL DEFAULT '',
    two_fa_enabled           BOOLEAN      NOT NULL DEFAULT TRUE,
    two_fa_required          BOOLEAN      NOT NULL DEFAULT FALSE,
    email_2fa_enabled        BOOLEAN      NOT NULL DEFAULT FALSE,
    two_fa_methods           VARCHAR(50)  NOT NULL DEFAULT 'totp',
    -- Passkey settings
    passkey_2fa_enabled      BOOLEAN      NOT NULL DEFAULT FALSE,
    passkey_login_enabled    BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Magic link
    magic_link_enabled       BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Login notifications
    login_notifications_enabled BOOLEAN   NOT NULL DEFAULT FALSE,
    suspicious_activity_alerts  BOOLEAN   NOT NULL DEFAULT FALSE,
    -- SMS 2FA
    sms_2fa_enabled          BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Trusted device
    trusted_device_enabled   BOOLEAN      NOT NULL DEFAULT FALSE,
    trusted_device_max_days  INTEGER      NOT NULL DEFAULT 30,
    -- Brute-force settings (NULL = use global defaults)
    bf_lockout_enabled       BOOLEAN,
    bf_lockout_threshold     INTEGER,
    bf_lockout_durations     VARCHAR(255),
    bf_lockout_window        VARCHAR(50),
    bf_lockout_tier_ttl      VARCHAR(50),
    bf_delay_enabled         BOOLEAN,
    bf_delay_start_after     INTEGER,
    bf_delay_max_seconds     INTEGER,
    bf_delay_tier_ttl        VARCHAR(50),
    bf_captcha_enabled       BOOLEAN,
    bf_captcha_site_key      VARCHAR(500),
    bf_captcha_secret_key    VARCHAR(500),
    bf_captcha_threshold     INTEGER,
    -- OIDC settings
    oidc_enabled             BOOLEAN      NOT NULL DEFAULT FALSE,
    oidc_rsa_private_key     TEXT         NOT NULL DEFAULT '',
    oidc_id_token_ttl        INTEGER      NOT NULL DEFAULT 3600,
    oidc_issuer_url          VARCHAR(500) NOT NULL DEFAULT '',
    -- Frontend URL override
    frontend_url             VARCHAR(500) NOT NULL DEFAULT '',
    -- Login page branding
    login_logo_url           VARCHAR(500) NOT NULL DEFAULT '',
    login_theme              VARCHAR(20)  NOT NULL DEFAULT 'auto',
    login_primary_color      VARCHAR(20)  NOT NULL DEFAULT '',
    login_secondary_color    VARCHAR(20)  NOT NULL DEFAULT '',
    login_display_name       VARCHAR(200) NOT NULL DEFAULT '',
    -- Password policy
    pw_min_length            INTEGER      NOT NULL DEFAULT 8,
    pw_max_length            INTEGER      NOT NULL DEFAULT 128,
    pw_require_upper         BOOLEAN      NOT NULL DEFAULT FALSE,
    pw_require_lower         BOOLEAN      NOT NULL DEFAULT FALSE,
    pw_require_digit         BOOLEAN      NOT NULL DEFAULT FALSE,
    pw_require_symbol        BOOLEAN      NOT NULL DEFAULT FALSE,
    pw_history_count         INTEGER      NOT NULL DEFAULT 0,
    pw_max_age_days          INTEGER      NOT NULL DEFAULT 0,
    -- Token TTL overrides (0 = use global env var defaults)
    access_token_ttl_minutes INTEGER      NOT NULL DEFAULT 0,
    refresh_token_ttl_hours  INTEGER      NOT NULL DEFAULT 0,
    -- Email action link path overrides
    reset_password_path      VARCHAR(500) NOT NULL DEFAULT '',
    magic_link_path          VARCHAR(500) NOT NULL DEFAULT '',
    verify_email_path        VARCHAR(500) NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── oauth_provider_configs ───────────────────────────────────────────────────

CREATE TABLE oauth_provider_configs (
    id           UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id       UUID    NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    provider     TEXT    NOT NULL,
    client_id    TEXT    NOT NULL,
    client_secret TEXT   NOT NULL,
    redirect_url TEXT    NOT NULL,
    is_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_app_provider ON oauth_provider_configs(app_id, provider);

-- ─── users ────────────────────────────────────────────────────────────────────

CREATE TABLE users (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id                 UUID         NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    email                  VARCHAR(255) NOT NULL,
    password_hash          TEXT         NOT NULL DEFAULT '',
    email_verified         BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active              BOOLEAN      NOT NULL DEFAULT TRUE,
    name                   TEXT         NOT NULL DEFAULT '',
    first_name             TEXT         NOT NULL DEFAULT '',
    last_name              TEXT         NOT NULL DEFAULT '',
    profile_picture        TEXT         NOT NULL DEFAULT '',
    locale                 TEXT         NOT NULL DEFAULT '',
    two_fa_enabled         BOOLEAN      NOT NULL DEFAULT FALSE,
    two_fa_method          VARCHAR(20)  NOT NULL DEFAULT '',
    two_fa_secret          TEXT         NOT NULL DEFAULT '',
    two_fa_recovery_codes  JSONB        NOT NULL DEFAULT '[]',
    backup_email           VARCHAR(255) NOT NULL DEFAULT '',
    backup_email_verified  BOOLEAN      NOT NULL DEFAULT FALSE,
    two_fa_previous_method VARCHAR(20)  NOT NULL DEFAULT '',
    two_fa_previous_secret TEXT         NOT NULL DEFAULT '',
    phone_number           VARCHAR(30)  NOT NULL DEFAULT '',
    phone_verified         BOOLEAN      NOT NULL DEFAULT FALSE,
    locked_at              TIMESTAMPTZ,
    lock_reason            VARCHAR(255) NOT NULL DEFAULT '',
    lock_expires_at        TIMESTAMPTZ,
    password_history       JSONB,
    password_changed_at    TIMESTAMPTZ,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_email_app_id ON users(email, app_id);

-- ─── social_accounts ──────────────────────────────────────────────────────────

CREATE TABLE social_accounts (
    id               UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id           UUID    NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    user_id          UUID    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT    NOT NULL,
    provider_user_id TEXT    NOT NULL,
    email            TEXT    NOT NULL DEFAULT '',
    name             TEXT    NOT NULL DEFAULT '',
    first_name       TEXT    NOT NULL DEFAULT '',
    last_name        TEXT    NOT NULL DEFAULT '',
    profile_picture  TEXT    NOT NULL DEFAULT '',
    username         TEXT    NOT NULL DEFAULT '',
    locale           TEXT    NOT NULL DEFAULT '',
    raw_data         JSONB,
    access_token     TEXT    NOT NULL DEFAULT '',
    refresh_token    TEXT    NOT NULL DEFAULT '',
    expires_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_provider_user_id_app_id ON social_accounts(app_id, provider, provider_user_id);
CREATE INDEX idx_social_accounts_app_id ON social_accounts(app_id);

-- ─── activity_logs ────────────────────────────────────────────────────────────

CREATE TABLE activity_logs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id     UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    timestamp  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address TEXT        NOT NULL DEFAULT '',
    user_agent TEXT        NOT NULL DEFAULT '',
    details    JSONB,
    severity   VARCHAR(20) NOT NULL DEFAULT 'INFORMATIONAL',
    expires_at TIMESTAMPTZ,
    is_anomaly BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_activity_logs_app_id        ON activity_logs(app_id);
CREATE INDEX idx_activity_logs_expires       ON activity_logs(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_activity_logs_cleanup       ON activity_logs(severity, expires_at, timestamp);
CREATE INDEX idx_activity_logs_user_timestamp ON activity_logs(user_id, timestamp DESC);

-- ─── operator_permissions ─────────────────────────────────────────────────────

CREATE TABLE operator_permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_operator_permission_resource_action UNIQUE (resource, action)
);

-- ─── operator_roles ───────────────────────────────────────────────────────────

CREATE TABLE operator_roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_operator_roles_name UNIQUE (name)
);

-- ─── operator_role_permissions ────────────────────────────────────────────────

CREATE TABLE operator_role_permissions (
    role_id       UUID NOT NULL REFERENCES operator_roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES operator_permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ─── admin_accounts ───────────────────────────────────────────────────────────

CREATE TABLE admin_accounts (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username              VARCHAR(255) NOT NULL,
    email                 VARCHAR(255),
    password_hash         VARCHAR(255) NOT NULL,
    operator_role_id      UUID         NOT NULL REFERENCES operator_roles(id) ON DELETE RESTRICT,
    two_fa_enabled        BOOLEAN      NOT NULL DEFAULT FALSE,
    two_fa_method         VARCHAR(20)           DEFAULT '',
    two_fa_secret         TEXT                  DEFAULT '',
    two_fa_recovery_codes JSONB                 DEFAULT '[]',
    magic_link_enabled    BOOLEAN      NOT NULL DEFAULT FALSE,
    backup_email          VARCHAR(255) NOT NULL DEFAULT '',
    backup_email_verified BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_login_at         TIMESTAMPTZ,
    disabled_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_admin_accounts_username         ON admin_accounts(username);
CREATE UNIQUE INDEX idx_admin_accounts_email            ON admin_accounts(email);
CREATE INDEX idx_admin_accounts_operator_role_id        ON admin_accounts(operator_role_id);

-- ─── api_keys ─────────────────────────────────────────────────────────────────

CREATE TABLE api_keys (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    key_type         VARCHAR(10) NOT NULL,
    name             VARCHAR(255) NOT NULL,
    description      TEXT        NOT NULL DEFAULT '',
    key_hash         VARCHAR(64) NOT NULL,
    key_prefix       VARCHAR(16) NOT NULL,
    key_suffix       VARCHAR(4)  NOT NULL,
    app_id           UUID        REFERENCES applications(id) ON DELETE CASCADE,
    scopes           TEXT        NOT NULL DEFAULT '',
    operator_role_id UUID        REFERENCES operator_roles(id) ON DELETE RESTRICT,
    expires_at       TIMESTAMPTZ,
    last_used_at     TIMESTAMPTZ,
    notified_7_days_at TIMESTAMPTZ,
    notified_1_day_at  TIMESTAMPTZ,
    is_revoked       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_api_keys_key_hash   ON api_keys(key_hash);
CREATE INDEX idx_api_keys_key_type          ON api_keys(key_type);
CREATE INDEX idx_api_keys_app_id            ON api_keys(app_id);
CREATE INDEX idx_api_keys_is_revoked        ON api_keys(is_revoked);
CREATE INDEX idx_api_keys_expires_at        ON api_keys(expires_at);
CREATE INDEX idx_api_keys_operator_role_id  ON api_keys(operator_role_id);
CREATE INDEX idx_api_keys_active_lookup     ON api_keys(key_hash, is_revoked) WHERE is_revoked = FALSE;

-- ─── api_key_usages ───────────────────────────────────────────────────────────

CREATE TABLE api_key_usages (
    id            BIGSERIAL   PRIMARY KEY,
    api_key_id    UUID        NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    period_date   DATE        NOT NULL,
    request_count BIGINT      NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_api_key_usage_key_period ON api_key_usages(api_key_id, period_date);
CREATE INDEX idx_api_key_usages_api_key_id       ON api_key_usages(api_key_id);

-- ─── operator_iam_events ──────────────────────────────────────────────────────

CREATE TABLE operator_iam_events (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    actor_kind         VARCHAR(32)  NOT NULL,
    actor_key_id       UUID         REFERENCES api_keys(id) ON DELETE SET NULL,
    actor_account_id   UUID         REFERENCES admin_accounts(id) ON DELETE SET NULL,
    target_kind        VARCHAR(32)  NOT NULL,
    target_key_id      UUID         REFERENCES api_keys(id) ON DELETE SET NULL,
    target_account_id  UUID         REFERENCES admin_accounts(id) ON DELETE SET NULL,
    old_role_id        UUID         REFERENCES operator_roles(id) ON DELETE RESTRICT,
    new_role_id        UUID         REFERENCES operator_roles(id) ON DELETE RESTRICT,
    action             VARCHAR(32)  NOT NULL,
    CONSTRAINT operator_iam_events_actor_kind_check CHECK (actor_kind IN ('env_key', 'api_key', 'gui_account', 'setup_cli')),
    CONSTRAINT operator_iam_events_target_kind_check CHECK (target_kind IN ('env_key', 'api_key', 'gui_account')),
    CONSTRAINT operator_iam_events_action_check CHECK (action IN ('assign', 'create_principal', 'revoke_key', 'disable_principal'))
);

CREATE INDEX idx_operator_iam_events_at ON operator_iam_events (at DESC);
CREATE INDEX idx_operator_iam_events_target_key_id ON operator_iam_events (target_key_id);
CREATE INDEX idx_operator_iam_events_target_account_id ON operator_iam_events (target_account_id);

-- ─── operator_access_logs ─────────────────────────────────────────────────────

CREATE TABLE operator_access_logs (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    kind        VARCHAR(32)  NOT NULL,
    key_id      UUID         REFERENCES api_keys(id) ON DELETE SET NULL,
    account_id  UUID         REFERENCES admin_accounts(id) ON DELETE SET NULL,
    role_name   VARCHAR(100) NOT NULL DEFAULT '',
    method      VARCHAR(16)  NOT NULL,
    path        TEXT         NOT NULL,
    decision    VARCHAR(16)  NOT NULL,
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    status      INTEGER      NOT NULL,
    CONSTRAINT operator_access_logs_kind_check CHECK (kind IN ('env_key', 'api_key', 'gui_account')),
    CONSTRAINT operator_access_logs_decision_check CHECK (decision IN ('allow', 'deny'))
);

CREATE INDEX idx_operator_access_logs_at ON operator_access_logs (at DESC);
CREATE INDEX idx_operator_access_logs_decision ON operator_access_logs (decision);

-- ─── system_settings ──────────────────────────────────────────────────────────

CREATE TABLE system_settings (
    key        VARCHAR(100) PRIMARY KEY,
    value      TEXT         NOT NULL DEFAULT '',
    category   VARCHAR(50)  NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_settings_category ON system_settings(category);

-- ─── email_server_configs ─────────────────────────────────────────────────────

CREATE TABLE email_server_configs (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id        UUID         REFERENCES applications(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL DEFAULT 'Default',
    smtp_host     VARCHAR(255) NOT NULL,
    smtp_port     INTEGER      NOT NULL DEFAULT 587,
    smtp_username VARCHAR(255)          DEFAULT '',
    smtp_password TEXT                  DEFAULT '',
    from_address  VARCHAR(255) NOT NULL,
    from_name     VARCHAR(100)          DEFAULT '',
    use_tls       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    is_default    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_server_configs_app_id              ON email_server_configs(app_id);
CREATE UNIQUE INDEX idx_email_server_configs_app_default  ON email_server_configs(app_id)  WHERE app_id IS NOT NULL AND is_default = TRUE;
CREATE UNIQUE INDEX idx_email_server_configs_global_default ON email_server_configs(is_default) WHERE app_id IS NULL AND is_default = TRUE;

-- ─── email_types ──────────────────────────────────────────────────────────────

CREATE TABLE email_types (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(50) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT                  DEFAULT '',
    default_subject VARCHAR(255)          DEFAULT '',
    variables       JSONB                 DEFAULT '[]',
    is_system       BOOLEAN     NOT NULL DEFAULT TRUE,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── email_templates ──────────────────────────────────────────────────────────

CREATE TABLE email_templates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          UUID        REFERENCES applications(id) ON DELETE CASCADE,
    email_type_id   UUID        NOT NULL REFERENCES email_types(id) ON DELETE CASCADE,
    server_config_id UUID       REFERENCES email_server_configs(id) ON DELETE SET NULL,
    name            VARCHAR(100) NOT NULL,
    subject         VARCHAR(255) NOT NULL,
    body_html       TEXT                  DEFAULT '',
    body_text       TEXT                  DEFAULT '',
    from_email      VARCHAR(255)          DEFAULT '',
    from_name       VARCHAR(255)          DEFAULT '',
    template_engine VARCHAR(20) NOT NULL DEFAULT 'go_template',
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_app_email_type    ON email_templates(app_id, email_type_id) WHERE app_id IS NOT NULL;
CREATE UNIQUE INDEX idx_global_email_type ON email_templates(email_type_id)         WHERE app_id IS NULL;

-- ─── permissions ──────────────────────────────────────────────────────────────

CREATE TABLE permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    description TEXT                  DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_permission_resource_action UNIQUE (resource, action)
);

-- ─── roles ────────────────────────────────────────────────────────────────────

CREATE TABLE roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      UUID         NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    description TEXT                  DEFAULT '',
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_role_app_name UNIQUE (app_id, name)
);

CREATE INDEX idx_roles_app_id ON roles(app_id);

-- ─── role_permissions ─────────────────────────────────────────────────────────

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ─── user_roles ───────────────────────────────────────────────────────────────

CREATE TABLE user_roles (
    user_id     UUID        NOT NULL REFERENCES users(id)        ON DELETE CASCADE,
    role_id     UUID        NOT NULL REFERENCES roles(id)        ON DELETE CASCADE,
    app_id      UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id   ON user_roles(user_id);
CREATE INDEX idx_user_roles_app_id    ON user_roles(app_id);
CREATE INDEX idx_user_role_app_user   ON user_roles(app_id, user_id);

-- ─── webhook_endpoints ────────────────────────────────────────────────────────

CREATE TABLE webhook_endpoints (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id     UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    url        TEXT        NOT NULL,
    secret     TEXT        NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_webhook_app_event         ON webhook_endpoints(app_id, event_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_webhook_endpoints_app_id         ON webhook_endpoints(app_id);
CREATE INDEX idx_webhook_endpoints_deleted_at     ON webhook_endpoints(deleted_at);

-- ─── webhook_deliveries ───────────────────────────────────────────────────────

CREATE TABLE webhook_deliveries (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id   UUID        NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    app_id        UUID        NOT NULL,
    event_type    VARCHAR(64) NOT NULL,
    payload       TEXT        NOT NULL,
    attempt       INT         NOT NULL DEFAULT 1,
    status_code   INT,
    response_body TEXT,
    latency_ms    BIGINT,
    success       BOOLEAN     NOT NULL DEFAULT FALSE,
    error_message TEXT,
    next_retry_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_endpoint_id ON webhook_deliveries(endpoint_id);
CREATE INDEX idx_webhook_deliveries_app_id      ON webhook_deliveries(app_id);
CREATE INDEX idx_webhook_deliveries_event_type  ON webhook_deliveries(event_type);
CREATE INDEX idx_webhook_deliveries_success     ON webhook_deliveries(success);
CREATE INDEX idx_webhook_deliveries_next_retry_at ON webhook_deliveries(next_retry_at) WHERE next_retry_at IS NOT NULL;

-- ─── oidc_clients ─────────────────────────────────────────────────────────────

CREATE TABLE oidc_clients (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id              UUID         NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name                VARCHAR(100) NOT NULL,
    description         TEXT         NOT NULL DEFAULT '',
    client_id           VARCHAR(64)  NOT NULL,
    client_secret_hash  TEXT         NOT NULL,
    redirect_uris       TEXT         NOT NULL DEFAULT '[]',
    allowed_grant_types VARCHAR(200) NOT NULL DEFAULT 'authorization_code,refresh_token',
    allowed_scopes      VARCHAR(200) NOT NULL DEFAULT 'openid profile email',
    require_consent     BOOLEAN      NOT NULL DEFAULT TRUE,
    is_confidential     BOOLEAN      NOT NULL DEFAULT TRUE,
    pkce_required       BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN      NOT NULL DEFAULT TRUE,
    logo_url            TEXT         NOT NULL DEFAULT '',
    login_theme         VARCHAR(20)  NOT NULL DEFAULT 'auto',
    login_primary_color VARCHAR(20)  NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_oidc_clients_client_id UNIQUE (client_id)
);

CREATE INDEX idx_oidc_clients_app_id        ON oidc_clients(app_id);
CREATE INDEX idx_oidc_clients_app_id_active ON oidc_clients(app_id, is_active);

-- ─── oidc_auth_codes ──────────────────────────────────────────────────────────

CREATE TABLE oidc_auth_codes (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id                UUID        NOT NULL,
    client_id             VARCHAR(64) NOT NULL,
    user_id               UUID        NOT NULL,
    code                  VARCHAR(128) NOT NULL,
    redirect_uri          TEXT        NOT NULL,
    scopes                TEXT        NOT NULL,
    nonce                 TEXT        NOT NULL DEFAULT '',
    code_challenge        TEXT        NOT NULL DEFAULT '',
    code_challenge_method VARCHAR(10) NOT NULL DEFAULT '',
    expires_at            TIMESTAMPTZ NOT NULL,
    used                  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_oidc_auth_codes_code UNIQUE (code)
);

CREATE INDEX idx_oidc_auth_codes_app_id    ON oidc_auth_codes(app_id);
CREATE INDEX idx_oidc_auth_codes_client_id ON oidc_auth_codes(client_id);
CREATE INDEX idx_oidc_auth_codes_expires_at ON oidc_auth_codes(expires_at);
CREATE INDEX idx_oidc_auth_codes_client_used ON oidc_auth_codes(client_id, used);

-- ─── trusted_devices ──────────────────────────────────────────────────────────

CREATE TABLE trusted_devices (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id      UUID         NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64)  NOT NULL,
    name        VARCHAR(255),
    user_agent  TEXT,
    ip_address  VARCHAR(45),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_trusted_devices_token_hash   ON trusted_devices(token_hash);
CREATE INDEX idx_trusted_device_user_app             ON trusted_devices(user_id, app_id);
CREATE INDEX idx_trusted_device_expires              ON trusted_devices(expires_at);

-- ─── webauthn_credentials ─────────────────────────────────────────────────────

CREATE TABLE webauthn_credentials (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID         REFERENCES users(id)         ON DELETE CASCADE,
    app_id           UUID         REFERENCES applications(id)  ON DELETE CASCADE,
    admin_id         UUID         REFERENCES admin_accounts(id) ON DELETE CASCADE,
    credential_id    BYTEA        NOT NULL,
    public_key       BYTEA        NOT NULL,
    attestation_type VARCHAR(50),
    aaguid           BYTEA,
    sign_count       INTEGER      NOT NULL DEFAULT 0,
    name             VARCHAR(100),
    transports       VARCHAR(255),
    backup_eligible  BOOLEAN      NOT NULL DEFAULT FALSE,
    backup_state     BOOLEAN      NOT NULL DEFAULT FALSE,
    last_used_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_webauthn_credentials_credential_id ON webauthn_credentials(credential_id);
CREATE INDEX idx_webauthn_credentials_user_id              ON webauthn_credentials(user_id);
CREATE INDEX idx_webauthn_credentials_app_id               ON webauthn_credentials(app_id);
CREATE INDEX idx_webauthn_credentials_admin_id             ON webauthn_credentials(admin_id);

-- ─── ip_rules ─────────────────────────────────────────────────────────────────

CREATE TABLE ip_rules (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      UUID         NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    rule_type   VARCHAR(10)  NOT NULL,
    match_type  VARCHAR(10)  NOT NULL,
    value       TEXT         NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ip_rule_app ON ip_rules(app_id);

-- ─── session_groups ───────────────────────────────────────────────────────────

CREATE TABLE session_groups (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name          TEXT    NOT NULL,
    description   TEXT    NOT NULL DEFAULT '',
    global_logout BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_groups_tenant_id ON session_groups(tenant_id);

-- ─── session_group_apps ───────────────────────────────────────────────────────

CREATE TABLE session_group_apps (
    session_group_id UUID        NOT NULL REFERENCES session_groups(id) ON DELETE CASCADE,
    app_id           UUID        NOT NULL REFERENCES applications(id)   ON DELETE CASCADE,
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_group_id, app_id)
);

CREATE UNIQUE INDEX idx_session_group_app_id ON session_group_apps(app_id);
