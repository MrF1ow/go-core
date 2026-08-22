# Phase 4: sqlc queries

[Overview](overview.md)

PR: Schema. Same PR as [phase 3](phase-3-schema.md). This is the only leftover phase that runs `sqlc generate`.

## Goal

Type-safe queries for events, access logs, account role, disable, and active superadmin count exist. Handlers still do not call them.

## Changes

- Extend `internal/queries/operator.sql` (or a sibling `internal/queries/operator_iam.sql` if the file would exceed the local size of `operator.sql`).
- Insert and list IAM events. Newest first. Optional filter by target key id or target account id. Limit plus offset.
- Insert and list access logs. Newest first. Optional filter by decision. Limit plus offset.
- `UpdateAdminAccountOperatorRole`, `SetAdminAccountDisabledAt`, `CountEnabledSuperadminAccounts` (role id plus `disabled_at IS NULL`).
- Run `sqlc generate`. Commit `internal/sqlcgen`.
- Keep `ListAllAdminAccounts` returning the new column once `schema.sql` has it.

Do not wire handlers. Wave 2 owns callers.

## Data structures

sqlc structs. Map to `pkg/models` only if a caller outside `internal/operator` needs them. Prefer keeping event and access types inside `internal/operator`.

## Verification

Static: `sqlc generate` is a no-op after commit (`git diff --stat internal/sqlcgen` empty). `go test -count=1 ./internal/operator ./internal/sqlcgen`.

Runtime: none.
