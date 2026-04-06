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
