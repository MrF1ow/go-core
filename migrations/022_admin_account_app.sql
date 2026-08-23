-- Migration: Optional application binding for admin accounts.
-- Depends on: 001_tenants_and_apps, 005_admin_accounts, 006_api_keys

ALTER TABLE admin_accounts
    ADD COLUMN app_id UUID REFERENCES applications(id) ON DELETE RESTRICT;

ALTER TABLE admin_accounts
    ADD CONSTRAINT admin_accounts_superadmin_is_platform
    CHECK (operator_role_id <> 'd0000000-0000-0000-0000-000000000001'::uuid OR app_id IS NULL);

UPDATE api_keys SET app_id = NULL WHERE key_type = 'admin' AND app_id IS NOT NULL;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_admin_app_id_null
    CHECK (key_type <> 'admin' OR app_id IS NULL);
