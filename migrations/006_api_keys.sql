-- Migration: Create api_keys and api_key_usages tables
-- Depends on: 005_admin_accounts (applications table must exist)

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
