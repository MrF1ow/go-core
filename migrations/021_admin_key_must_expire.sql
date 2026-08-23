-- Migration: Require expires_at on admin API keys.
-- Depends on: 006_api_keys

UPDATE api_keys
SET expires_at = NOW() + INTERVAL '365 days'
WHERE key_type = 'admin' AND expires_at IS NULL;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_admin_must_expire
    CHECK (key_type <> 'admin' OR expires_at IS NOT NULL);
