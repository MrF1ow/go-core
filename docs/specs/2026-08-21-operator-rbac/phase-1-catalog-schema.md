# Phase 1: Catalog schema

[Overview](overview.md)

## Goal

Operator IAM tables exist and SQLC can read them. No request behavior changes.

## Changes

- Add `migrations/016_operator_rbac.sql`: `operator_permissions`, `operator_roles`, `operator_role_permissions`. Unique `(resource, action)` and unique `operator_roles.name`. `is_system` on roles.
- Add nullable `operator_role_id` on `api_keys` (FK to `operator_roles`, ON DELETE RESTRICT). App keys leave it null.
- Mirror the final shape in `internal/schema.sql`.
- Add `internal/queries/operator.sql` and models under `pkg/models/` (`OperatorPermission`, `OperatorRole`).
- Run `sqlc generate`.
- Package `internal/operator` can be a stub that compiles.

Do not seed yet. Do not alter `admin_accounts`. Do not parse this in middleware.

## Data structures

`OperatorPermission` is `Resource string` + `Action string`. `OperatorRole` is `Name`, `Description`, `IsSystem`, and a list of permissions. Join table is `(role_id, permission_id)` only.

## Verification

Static: `sqlc generate` succeeds. `go test ./internal/sqlcgen` if present, otherwise compile `./internal/operator`.

Runtime: `make migrate-up` against local Postgres applies `016_` (or document that CI migration tests cover `RunCoreMigrations`). No HTTP test. Flag: no control skill for SQL apply in this environment beyond `cmd/migrate`.
