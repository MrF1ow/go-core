-- Migration: Initial base schema
-- Description: Creates the core tables that existed before the migration system
-- was adopted. Uses IF NOT EXISTS so existing databases are unaffected.

-- ─── users ────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email                  VARCHAR(255) NOT NULL,
    password_hash          TEXT         NOT NULL DEFAULT '',
    email_verified         BOOLEAN      NOT NULL DEFAULT FALSE,
    name                   TEXT         NOT NULL DEFAULT '',
    first_name             TEXT         NOT NULL DEFAULT '',
    last_name              TEXT         NOT NULL DEFAULT '',
    profile_picture        TEXT         NOT NULL DEFAULT '',
    locale                 TEXT         NOT NULL DEFAULT '',
    two_fa_enabled         BOOLEAN      NOT NULL DEFAULT FALSE,
    two_fa_secret          TEXT         NOT NULL DEFAULT '',
    two_fa_recovery_codes  JSONB        NOT NULL DEFAULT '[]',
    backup_email           VARCHAR(255) NOT NULL DEFAULT '',
    backup_email_verified  BOOLEAN      NOT NULL DEFAULT FALSE,
    phone_number           VARCHAR(30)  NOT NULL DEFAULT '',
    phone_verified         BOOLEAN      NOT NULL DEFAULT FALSE,
    locked_at              TIMESTAMPTZ,
    lock_reason            VARCHAR(255) NOT NULL DEFAULT '',
    lock_expires_at        TIMESTAMPTZ,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- ─── social_accounts ──────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS social_accounts (
    id               UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_user_id ON social_accounts(provider, provider_user_id);

-- ─── activity_logs ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS activity_logs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    timestamp  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address TEXT        NOT NULL DEFAULT '',
    user_agent TEXT        NOT NULL DEFAULT '',
    details    JSONB
);
