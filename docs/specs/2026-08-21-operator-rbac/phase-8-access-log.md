# Phase 8: Access log

[Overview](overview.md)

## Goal

Every admin JSON (and later GUI) permission decision can be listed: principal, role, method, path, allow or deny. Env key use is visible.

## Changes

- Table `operator_access_logs`: `id`, `at`, `kind`, `key_id`, `account_id`, `role_name`, `method`, `path`, `decision` (`allow`/`deny`), `resource`, `action`, `status`.
- Write from `RequireOperatorPermission` after the decision, best-effort, do not fail the request if insert fails. Log the insert error.
- Cap volume: log denies always. Log allows for `write` always. Log `read` allows sampled or skip in v1 if volume is a concern. **Experience First:** denies and env-key allows always. Env-key allow is the break-glass signal.
- JSON `GET /admin/operator/access-logs` with `admin_iam:read`. Retention: do not attach this to the short end-user informational TTL. Use a dedicated longer default (for example 365 days) or no expiry in v1 with a documented cleanup later.
- Tests: 403 writes `deny`. Env allow on `tenants:write` writes `allow` with `kind=env_key`.

## Data structures

One row per checked request (with the read-allow exception above). Principal ids nullable for env key.

## Verification

Static: `go test -count=1 ./internal/middleware ./internal/operator`.

Runtime: httptest deny then GET access-logs as superadmin. Row matches path and `deny`.

No control-ui.
