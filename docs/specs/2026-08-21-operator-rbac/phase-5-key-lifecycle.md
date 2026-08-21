# Phase 5: Key lifecycle

[Overview](overview.md)

## Goal

Admin API keys have a real role. New admin keys are `viewer`. Existing admin keys with blank `scopes` are `superadmin`. The GUI form assigns a role, not a freeform scope string, for `key_type=admin`.

Keys may expire. Default is forever. An operator who wants a time-boxed key sets `expires_at`. After that time the key dies on its own. Do not invent a second expiry mechanism. The column, create form, `FindActiveKeyByHash`, and expiry emails already exist.

## Changes

- Backfill SQL: `UPDATE api_keys SET operator_role_id = (SELECT id FROM operator_roles WHERE name = 'superadmin') WHERE key_type = 'admin' AND operator_role_id IS NULL`. Existing `expires_at` values stay as they are (usually null).
- Create-key path (`internal/admin` GUI handler and any JSON create if it exists): admin keys default `operator_role_id` to `viewer`. Reject unknown role ids. Creating a key with `superadmin` or `admin_iam` requires the caller to have `admin_iam:write` (after phase 4, only env and superadmin keys). Until GUI accounts exist, that means env key or a superadmin DB key.
- Edit-key path: change role, not `scopes`, for admin keys. App keys keep `scopes` unused as today.
- Keep optional expiry on create. Empty field = forever (`expires_at` null). Copy should say that, not only "(optional)".
- Add expiry to the edit form. Today create has `datetime-local` `expires_at`; `api_key_edit_form.tmpl` and `UpdateApiKeyScopes` do not. An operator must be able to set a date, change it, or clear it back to forever.
- Optional UX: duration presets (forever, 30 days, 90 days, 1 year, custom datetime) that write `expires_at`. Default selection is forever. That is the expire-automatically control. The stored fact is still one timestamp or null.
- Keep the 7-day and 1-day expiry emails. They already skip null `expires_at`.
- Templates `web/templates/partials/api_key_form.tmpl` and `api_key_edit_form.tmpl`: role select for admin keys. Stop telling operators that blank scopes are unrestricted. Do not drop the expiry field when scopes become a role select.
- Leave `scopes` column in place. Stop writing it for new admin keys (empty string). **Subtract Before You Add** on the authz path only.
- Tests: default role on create; backfill query; form no longer contains the old unrestricted copy. Create with no expiry stores null. Create with a future timestamp stores it. Edit can clear expiry. Expired key is 401 (covered in phase 3; keep a create/edit round-trip here).

## Data structures

`api_keys.operator_role_id` is the grant. `scopes` is leftover storage, not authorization. `expires_at` is optional lifetime, independent of the role.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/operator`.

Runtime: create-key handler test with a stub repo asserting `viewer` and null `ExpiresAt` when the form omits the date. Template test or string assert on rendered form for role `<select>` and an expiry control on both create and edit.
