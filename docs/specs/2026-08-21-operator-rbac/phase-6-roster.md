# Phase 6: Roster

[Overview](overview.md)

## Goal

An operator with `admin_iam:read` can list every admin key and (once phase 9 exists) every GUI account with current role, created at, last used, and key expiry. Export exists for access reviews.

## Changes

- Service method listing principals: admin keys (`id`, `name`, `key_prefix`, `role`, `created_at`, `last_used_at`, `expires_at`, `revoked`) and a placeholder empty account list until phase 9. Null `expires_at` displays as forever. An expired key stays on the roster until revoked or deleted so a review can see it died on schedule. The GUI key list already has an Expires column. Keep that.
- JSON `GET /admin/operator/roster` behind `RequireOperatorPermission("admin_iam", "read")`.
- JSON `GET /admin/operator/roster/export` same permission, CSV and/or JSON. Cap rows like activity-log export.
- Tests: viewer key 403. Superadmin 200 with at least the env key represented as a synthetic row (`kind=env_key`, no id) so reviewers see break-glass exists.

Do not log every roster fetch as IAM history. Access log (phase 8) covers the HTTP call.

## Data structures

`RosterEntry`: `Kind`, optional `KeyID`, optional `AccountID`, `DisplayName`, `RoleName`, `CreatedAt`, `LastUsedAt`, `ExpiresAt` (nil = forever; env key has none), `Revoked`. Env key is one synthetic entry, not a table row.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/coreapp`.

Runtime: httptest GET roster with viewer → 403. With superadmin → body contains `viewer` and `superadmin` rows after fixtures.
