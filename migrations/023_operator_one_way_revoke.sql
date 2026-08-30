-- Migration: One-way revoke and disable. Rows stay; credentials cannot come back.
-- Depends on: 006_api_keys, 018_operator_iam_evidence

CREATE OR REPLACE FUNCTION prevent_admin_account_reenable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.disabled_at IS NOT NULL AND NEW.disabled_at IS NULL THEN
        RAISE EXCEPTION 'disabled_at cannot be cleared';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER admin_accounts_disabled_at_one_way
    BEFORE UPDATE ON admin_accounts
    FOR EACH ROW
    EXECUTE FUNCTION prevent_admin_account_reenable();

CREATE OR REPLACE FUNCTION prevent_api_key_unrevoke()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.is_revoked AND NOT NEW.is_revoked THEN
        RAISE EXCEPTION 'is_revoked cannot be cleared';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER api_keys_is_revoked_one_way
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION prevent_api_key_unrevoke();
