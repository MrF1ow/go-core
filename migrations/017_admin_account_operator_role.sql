ALTER TABLE admin_accounts
    ADD COLUMN operator_role_id UUID REFERENCES operator_roles(id) ON DELETE RESTRICT;

-- Keep this backfill in sync with internal/operator/seed_ids.go RoleIDSuperadmin.
UPDATE admin_accounts SET operator_role_id = 'd0000000-0000-0000-0000-000000000001' WHERE operator_role_id IS NULL;

ALTER TABLE admin_accounts
    ALTER COLUMN operator_role_id SET NOT NULL;

CREATE INDEX idx_admin_accounts_operator_role_id ON admin_accounts(operator_role_id);
