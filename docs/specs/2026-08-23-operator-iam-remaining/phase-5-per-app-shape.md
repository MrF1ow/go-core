# Phase 5: Per-app shape

[Overview](overview.md)

PR: Per-app shape. Base is Must-expire. Exclusive `sqlc generate`.

## Goal

A GUI operator can be bound to one application. Superadmin cannot. Platform operators keep null `app_id`. `Principal` carries that binding.

## Changes

- Add `migrations/022_admin_account_app.sql`.
- `admin_accounts.app_id UUID REFERENCES applications(id) ON DELETE RESTRICT`, nullable.
- CHECK that superadmin accounts have `app_id IS NULL`. Use `operator_roles` name or the seeded superadmin UUID already pinned in Go.
- Update `internal/schema.sql`. List/get account queries return `app_id`. Exclusive `sqlc generate`.
- `Principal.AppID *uuid.UUID`. Nil is platform. Set is that app only.
- Account create JSON and GUI stay platform viewer with null `app_id` in this phase. Do not add the app picker yet.
- Last-superadmin count is enabled superadmins with `app_id IS NULL`.
- Pin `022`.

Do not deny platform routes yet. Do not filter lists yet. A bound account that still reaches Tenants is expected until phase 7.

## Data structures

`Principal.AppID *uuid.UUID`. Nil means platform. `WouldLeaveLastSuperadmin` still takes a count. The count query is what changes.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/sqlcgen`.

Runtime: insert account with superadmin role and a non-null `app_id` fails the CHECK. Insert viewer with an app_id succeeds. Last-superadmin still 409s a platform superadmin demote.
