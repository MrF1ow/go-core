# Phase 4: JSON route enforcement

[Overview](overview.md)

## Goal

Every admin JSON route declares a permission. A viewer DB key cannot create a tenant. The env key still can. Admin OIDC JSON and `GET /metrics` are included. GUI is unchanged.

## Changes

- `internal/coreapp/app.go`: on `adminRoutes`, `metricsGroup`, and `adminOIDC`, add `RequireOperatorPermission` per the overview HTTP mapping.
- Resource mapping (JSON):
  - `/admin/tenants` → `tenants`
  - `/admin/apps` (except nested) → `applications`
  - `/admin/apps/:id/oauth-config` → `oauth`
  - `/admin/oidc/...` → `oidc`
  - `/admin/apps/:id/ip-rules` and check → `ip_rules`
  - `/admin/rbac/...` → `end_user_rbac`
  - `/admin/email-*`, `/admin/apps/:id/email-*`, send-email, preview → `email`
  - `/admin/webhooks` → `webhooks`
  - `/admin/users/export|import` and trusted-devices → `users`
  - `/admin/activity-logs` → `logs` (`read`)
  - `/metrics` → `monitoring` (`read`)
- Session groups have no JSON admin group today. Skip until GUI phase or add later. Do not invent JSON routes.
- Integration tests in `internal/coreapp` or `internal/middleware` using a test Gin engine: viewer key GET activity-logs 200 (if the handler is stubbed) or middleware-only if full DB is too heavy. Minimum: one real route group with stub handlers proving 403 vs 200.
- **how** this subsystem before editing `RegisterRoutes`. **interrogate** the mapping if a nested path is ambiguous (`/admin/apps/:id/email-config` is `email`, not `applications`).

Ship with phase 5 backfill in the same PR unless `016`/`017` already set every admin key’s `operator_role_id`.

## Data structures

No new types. Route table is the grant surface. Do not hide checks inside individual handlers.

## Verification

Static: `go test -count=1 ./internal/coreapp ./internal/middleware ./internal/oidc`.

Runtime: httptest `POST /admin/tenants` with viewer principal → 403, handler not called. Same with env principal → handler called. `GET /metrics` with viewer (has `monitoring:read`) → allowed. `GET /metrics` with a custom role lacking it → 403.

No control-ui. Flag: full Postgres-backed handler tests only if `make test` already has that harness.
