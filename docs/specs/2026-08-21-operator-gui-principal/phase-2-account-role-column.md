# Phase 2: Account role column

[Overview](overview.md)

## Goal

Every `admin_accounts` row has an operator role FK. Existing rows become superadmin. The column is NOT NULL after backfill.

## Changes

- Migration `017_admin_account_operator_role.sql`. Add `admin_accounts.operator_role_id UUID REFERENCES operator_roles(id) ON DELETE RESTRICT`. Backfill `WHERE operator_role_id IS NULL` to `d0000000-0000-0000-0000-000000000001`. Then `SET NOT NULL`. Index like `api_keys`.
- Mirror DDL in `internal/schema.sql`. Run `sqlc generate`.
- Extend `models.AdminAccount` and `AccountRepository` create/lookup mapping. Create requires a role ID. Do not default in SQL; the caller chooses.
- Drop the “full system access” comment on `AdminAccount`. That sentence is no longer true after phase 4, and it is already false for JSON.

## Data structures

`AdminAccount.OperatorRoleID uuid.UUID` (value, not pointer, after NOT NULL). Same seed UUIDs as keys.

## Verification

Static: `gofmt`, `sqlc generate` diff empty after a second run, `go test -count=1 ./internal/admin`.

Runtime: repository test with a fake or testdb: insert without a role fails; insert with viewer UUID round-trips.
