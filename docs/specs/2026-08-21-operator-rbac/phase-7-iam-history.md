# Phase 7: IAM history

[Overview](overview.md)

## Goal

Role assignment is an append-only event. You can see who changed which principal from which role to which role.

## Changes

- Table `operator_iam_events`: `id`, `at`, `actor_kind`, `actor_key_id`, `actor_account_id`, `target_kind`, `target_key_id`, `target_account_id`, `old_role_id`, `new_role_id`, `action` (`assign`, `create_principal`, `revoke_key`).
- Write a row whenever an admin key’s role changes, a key is created, or (phase 9) an account’s role changes. Actor is the `OperatorPrincipal` on the request. Setup CLI uses `actor_kind=setup_cli` with null actor ids.
- JSON `GET /admin/operator/iam-events` with `admin_iam:read`. Filter by target id. Newest first. Same export cap pattern.
- Assign-role JSON `PUT /admin/operator/keys/:id/role` with `admin_iam:write`. Last-superadmin rule applies to GUI accounts in phase 9, not to keys (keys may all be viewer).
- Tests: assign writes old and new role ids; list returns them; viewer cannot PUT.

## Data structures

Append-only events. No updates. No deletes except via retention job later (out of scope unless cleanup already has a hook). Do not reuse `activity_logs`.

## Verification

Static: `go test -count=1 ./internal/operator`.

Runtime: httptest assign then list. Assert actor kind `api_key` and new role `support`.
