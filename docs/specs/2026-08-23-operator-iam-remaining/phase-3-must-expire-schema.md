# Phase 3: Must-expire schema

[Overview](overview.md)

PR: Must-expire. Base is Access client.

## Goal

An admin API key cannot have a null `expires_at`. Existing null admin keys get a finite instant. App keys stay nullable. The env key has no row.

## Changes

- Add `migrations/021_admin_key_must_expire.sql`.
- `UPDATE api_keys SET expires_at = NOW() + INTERVAL '365 days' WHERE key_type = 'admin' AND expires_at IS NULL`.
- `CHECK (key_type <> 'admin' OR expires_at IS NOT NULL)`.
- Update `internal/schema.sql`. No new sqlc queries. Do not run `sqlc generate` unless a query actually changes.
- Pin `021` in `catalog_sql_test.go`.

Do not change the GUI parser yet. A raw insert of a null admin expiry must fail after migrate.

## Data structures

No new Go type. `expires_at` on admin keys is a `time.Time` in practice. The pointer stays on the model because app keys still omit it.

## Verification

Static: `go test -count=1 ./internal/operator`.

Runtime: test migration or a CHECK assertion. Insert admin key with null `expires_at` fails. Insert app key with null `expires_at` succeeds.
