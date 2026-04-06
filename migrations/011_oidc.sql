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
