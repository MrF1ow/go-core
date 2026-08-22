# Phase 1: Fail-open schema

[Overview](overview.md)

PR: Fail-open. Lands with [phase 2](phase-2-fail-open-auth.md). Exclusive `schema.sql` writer. No `sqlc generate`.

## Goal

Admin keys cannot store a null role. App keys still can. Leftover null admin rows become viewer before the CHECK lands.

## Changes

- Add `migrations/019_admin_key_operator_role_required.sql`.
- Update `internal/schema.sql` to match.
- `UPDATE api_keys SET operator_role_id = 'd0000000-0000-0000-0000-000000000004' WHERE key_type = 'admin' AND operator_role_id IS NULL`. Viewer, not superadmin. New creates already stamp viewer. Remaining nulls are bugs, not the 016 existing-key backfill.
- `ALTER TABLE api_keys ADD CONSTRAINT api_keys_admin_operator_role_required CHECK (key_type <> 'admin' OR operator_role_id IS NOT NULL)`.
- Pin the new file in `internal/operator/catalog_sql_test.go` the same way `018` is pinned. Assert the CHECK text and the viewer id in the UPDATE.
- Do not `SET NOT NULL` on `operator_role_id`. App keys stay null.

Do not change `principalForDBKey` here. That is phase 2. A CHECK without the auth cutover still fail-opens rows that sneak in through a writer that ignores the CHECK (tests with in-memory keys).

## Data structures

No new Go types. The CHECK is the type.

## Verification

Static: `go test -count=1 ./internal/operator`. Migration pin test red then green.

Runtime: none until phase 2.
