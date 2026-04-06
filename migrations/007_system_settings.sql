-- Migration: Create system_settings table
-- Depends on: 006_api_keys

-- ─── system_settings ──────────────────────────────────────────────────────────

CREATE TABLE system_settings (
    key        VARCHAR(100) PRIMARY KEY,
    value      TEXT         NOT NULL DEFAULT '',
    category   VARCHAR(50)  NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_settings_category ON system_settings(category);
