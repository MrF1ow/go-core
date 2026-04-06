-- Migration: 005_admin_accounts
-- Description: Create admin_accounts table

CREATE TABLE admin_accounts (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username              VARCHAR(255) NOT NULL,
    email                 VARCHAR(255),
    password_hash         VARCHAR(255) NOT NULL,
    two_fa_enabled        BOOLEAN      NOT NULL DEFAULT FALSE,
    two_fa_method         VARCHAR(20)           DEFAULT '',
    two_fa_secret         TEXT                  DEFAULT '',
    two_fa_recovery_codes JSONB                 DEFAULT '[]',
    magic_link_enabled    BOOLEAN      NOT NULL DEFAULT FALSE,
    backup_email          VARCHAR(255) NOT NULL DEFAULT '',
    backup_email_verified BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_login_at         TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_admin_accounts_username ON admin_accounts(username);
CREATE UNIQUE INDEX idx_admin_accounts_email    ON admin_accounts(email);
