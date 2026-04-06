-- Migration: 002_users
-- Description: Create users table

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
