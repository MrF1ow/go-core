-- Migration: 003_social_accounts
-- Description: Create social_accounts table

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
