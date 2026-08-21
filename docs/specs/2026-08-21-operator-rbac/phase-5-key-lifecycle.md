# Phase 5: Key lifecycle

[Overview](overview.md)

## Goal

Admin API keys have a real role. New admin keys are `viewer`. Existing admin keys with blank `scopes` are `superadmin`. The GUI form assigns a role, not a freeform scope string, for `key_type=admin`.

## Changes

- Backfill SQL: `UPDATE api_keys SET operator_role_id = (SELECT id FROM operator_roles WHERE name = 'superadmin') WHERE key_type = 'admin' AND operator_role_id IS NULL`.
- Create-key path (`internal/admin` GUI handler and any JSON create if it exists): admin keys default `operator_role_id` to `viewer`. Reject unknown role ids. Creating a key with `superadmin` or `admin_iam` requires the caller to have `admin_iam:write` (after phase 4, only env and superadmin keys). Until GUI accounts exist, that means env key or a superadmin DB key.
- Edit-key path: change role, not `scopes`, for admin keys. App keys keep `scopes` unused as today.
- Templates `web/templates/partials/api_key_form.tmpl` and `api_key_edit_form.tmpl`: role select for admin keys. Stop telling operators that blank scopes are unrestricted.
- Leave `scopes` column in place. Stop writing it for new admin keys (empty string). **Subtract Before You Add** on the authz path only.
- Tests: default role on create; backfill query; form no longer contains the old unrestricted copy.

## Data structures

`api_keys.operator_role_id` is the grant. `scopes` is leftover storage, not authorization.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/operator`.

Runtime: create-key handler test with a stub repo asserting `viewer`. Template test or string assert on rendered form for role `<select>`.
