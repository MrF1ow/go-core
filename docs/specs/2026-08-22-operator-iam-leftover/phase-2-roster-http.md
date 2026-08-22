# Phase 2: Roster HTTP

[Overview](overview.md)

PR: Roster. Depends on [phase 1](phase-1-roster-types.md) in the same PR.

## Goal

An `admin_iam:read` principal can list and export the roster over JSON.

## Changes

- New JSON handler in `internal/operator/handler.go`. Do not grow `internal/admin/handler.go`.
- Wire in `internal/coreapp/app.go` on `adminRoutes` with `requireOp(operator.ResAdminIAM, operator.ActionRead)`. Stay on `adminRoutes` so `require_op_inventory_test.go` keeps scanning the new lines. Do not add a second group the inventory does not know.
- `GET /admin/operator/roster` returns JSON `{ "entries": [...] }`.
- `GET /admin/operator/roster/export` returns CSV. Same rows. Same cap. Filename `operator-roster.csv`.
- Fetch keys via existing `AdminListApiKeys` with `key_type=admin` and a limit of `ExportMaxRows`. Fetch accounts via `ListAllAdminAccounts`.
- Viewer key 403. Superadmin 200. Body includes `kind=env_key` and at least one fixture account or key role name.

Do not add GUI routes. Do not add a nav row.

## Data structures

Reuse `RosterEntry`. CSV columns: `kind,id,display_name,role,created_at,last_used_at,expires_at,revoked`. `id` is key UUID, account UUID, or empty for env.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/coreapp`.

Runtime: httptest GET roster with viewer → 403. With superadmin → body contains `env_key` and `superadmin`.
