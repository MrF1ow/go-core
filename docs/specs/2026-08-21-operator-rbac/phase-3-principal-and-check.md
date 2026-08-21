# Phase 3: Principal and check

[Overview](overview.md)

## Goal

Every authenticated admin JSON request has an `OperatorPrincipal` on the Gin context. `RequireOperatorPermission` can allow or deny. Routes are not wired yet except in tests.

## Changes

- Add `OperatorPrincipal` in `internal/operator` (or `pkg/models` if handlers outside operator need it without importing internal cycles). Kind is `env_key`, `api_key`, or `gui_account`. Optional key id, optional account id, role name, permission set.
- `AdminAuthMiddleware` after a successful env match: set principal with kind `env_key`, role name `superadmin`, permission set = all catalog actions (in memory, no DB). Still set `auth_type=admin`.
- After a successful DB admin key: load `operator_role_id`. If null, treat as `superadmin` for this phase so existing rows do not 403 when phase 4 lands in the same release. Load that role’s permissions. Set principal. Stop using `parseScopes` for admin keys.
- An expired DB key (`expires_at` in the past) is not a principal. `FindActiveKeyByHash` already returns nil. Auth stays 401, same as unknown or revoked. Do not attach a grant and 403. The key is dead. Env key has no expiry.
- Add `RequireOperatorPermission(resource, action)` in `internal/middleware`. Missing principal is 500 (auth middleware bug). Missing permission is 403 JSON. Env principal is not a special case inside this function.
- Tests in `internal/middleware`: env principal can `tenants:write`; viewer principal cannot; missing context key denies; type-assert failure denies. Expired hashed key: middleware 401, no principal on context.

Do not register the new middleware on `adminRoutes` in this phase if phase 4 is a separate PR. If 3 and 4 ship together, skip the “unwired” window.

## Data structures

`OperatorPrincipal` with a `Has(resource, action string) bool` method. Permission set is a map keyed by `resource:action`. Wildcard is not in v1. Roles are closed sets.

## Verification

Static: `go test -count=1 ./internal/middleware ./internal/operator`.

Runtime: httptest of middleware only, not full `/admin/tenants`. Prove env principal and viewer principal through `RequireOperatorPermission`. Prove a key with past `expires_at` never attaches a principal.
