-- Migration: Require operator_role_id on admin API keys.
-- Depends on: 016_operator_rbac

UPDATE api_keys
SET operator_role_id = 'd0000000-0000-0000-0000-000000000004'
WHERE key_type = 'admin' AND operator_role_id IS NULL;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_admin_operator_role_required
    CHECK (key_type <> 'admin' OR operator_role_id IS NOT NULL);
