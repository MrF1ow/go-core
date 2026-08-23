# Phase 1: Access-log client schema

[Overview](overview.md)

PR: Access client. Lands with [phase 2](phase-2-access-client-http.md).

## Goal

`operator_access_logs` can store client IP and user agent. Existing rows stay valid as empty strings.

## Changes

- Add `migrations/020_operator_access_log_client.sql`.
- Columns `ip_address TEXT NOT NULL DEFAULT ''` and `user_agent TEXT NOT NULL DEFAULT ''`, same types as `activity_logs`.
- Update `internal/schema.sql` in this PR.
- Extend insert and list SQL. This PR is the exclusive `sqlc generate` writer.
- Pin `020` in `internal/operator/catalog_sql_test.go`.
- Map the two fields on `AccessRecord`.

Do not capture from gin yet. Do not change CSV or templates yet.

## Data structures

`AccessRecord` gains `IPAddress string` and `UserAgent string`. JSON names `ip_address` and `user_agent`. Empty means unknown, not null.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/sqlcgen`. `git diff --stat internal/sqlcgen` is the intended generate output.

Runtime: none until phase 2. Insert with empty strings must succeed against the new columns.
