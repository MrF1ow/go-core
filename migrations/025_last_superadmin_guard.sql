-- Migration: Last enabled platform superadmin cannot be cleared.
-- Depends on: 022_admin_account_app, 023_operator_one_way_revoke

CREATE OR REPLACE FUNCTION prevent_last_superadmin_cleared()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    remaining int;
BEGIN
    IF (OLD.operator_role_id = 'd0000000-0000-0000-0000-000000000001'::uuid
        AND OLD.disabled_at IS NULL
        AND OLD.app_id IS NULL)
       AND NOT (NEW.operator_role_id = 'd0000000-0000-0000-0000-000000000001'::uuid
        AND NEW.disabled_at IS NULL
        AND NEW.app_id IS NULL)
    THEN
        PERFORM pg_advisory_xact_lock(
            hashtextextended('admin_accounts.last_enabled_superadmin', 0)
        );
        SELECT COUNT(*) INTO remaining FROM admin_accounts
        WHERE id <> NEW.id
          AND operator_role_id = 'd0000000-0000-0000-0000-000000000001'::uuid
          AND disabled_at IS NULL
          AND app_id IS NULL;
        IF remaining = 0 THEN
            RAISE EXCEPTION 'cannot demote or disable the last enabled superadmin';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER admin_accounts_last_superadmin
    BEFORE UPDATE ON admin_accounts
    FOR EACH ROW
    EXECUTE FUNCTION prevent_last_superadmin_cleared();
