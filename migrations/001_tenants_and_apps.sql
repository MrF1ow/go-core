-- Migration: 001_tenants_and_apps
-- Description: Create tenants, applications, and oauth_provider_configs tables with seed data

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

-- ─── seed data ───────────────────────────────────────────────────────────────

INSERT INTO tenants (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Tenant');
INSERT INTO applications (id, tenant_id, name, description) VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Default App', 'Default application');
