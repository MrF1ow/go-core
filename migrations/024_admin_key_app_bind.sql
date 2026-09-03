ALTER TABLE api_keys
    DROP CONSTRAINT api_keys_admin_app_id_null;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_admin_superadmin_is_platform
    CHECK (key_type <> 'admin' OR operator_role_id <> 'd0000000-0000-0000-0000-000000000001'::uuid OR app_id IS NULL);

CREATE OR REPLACE FUNCTION prevent_delete_app_with_admin_keys()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM api_keys
        WHERE key_type = 'admin' AND app_id = OLD.id
    ) THEN
        RAISE EXCEPTION 'application has bound admin keys';
    END IF;
    RETURN OLD;
END;
$$;

CREATE TRIGGER applications_admin_keys_restrict
    BEFORE DELETE ON applications
    FOR EACH ROW
    EXECUTE FUNCTION prevent_delete_app_with_admin_keys();
