# Phase 3: Schema 018

[Overview](overview.md)

PR: Schema. Lands with [phase 4](phase-4-sqlc.md). Exclusive `sqlc generate` writer.

## Goal

Operator evidence tables and `disabled_at` exist. No handlers yet.

## Changes

- Add `migrations/018_operator_iam_evidence.sql`.
- Update `internal/schema.sql` to match. sqlc reads this file.
- `operator_iam_events`: `id`, `at`, `actor_kind`, `actor_key_id`, `actor_account_id`, `target_kind`, `target_key_id`, `target_account_id`, `old_role_id`, `new_role_id`, `action`. Action check: `assign`, `create_principal`, `revoke_key`, `disable_principal`. Nullable actor and target ids.
- `operator_access_logs`: `id`, `at`, `kind`, `key_id`, `account_id`, `role_name`, `method`, `path`, `decision`, `resource`, `action`, `status`. Decision check: `allow`, `deny`.
- `admin_accounts.disabled_at TIMESTAMPTZ` nullable. Null means enabled.
- Indexes on `operator_iam_events(at DESC)` and `operator_access_logs(at DESC)`.
- No `IF NOT EXISTS` on new tables.
- Pin the new file in `internal/operator/catalog_sql_test.go` so a missing `018` fails like `016` and `017`.

Do not register routes. Do not hook middleware.

## Data structures

Append-only events and access rows. No updates. No deletes in v1.

`disabled_at` is the enabled/disabled flag. Do not add `is_disabled`.

## Verification

Static: `go test -count=1 ./internal/operator`. Migration pin test red then green.

Runtime: none until phase 4 generate.
