# Phase 2: Fail-open auth

[Overview](overview.md)

PR: Fail-open. Depends on [phase 1](phase-1-fail-open-schema.md) in the same PR.

## Goal

A DB admin key with a null role is 401. It is not in-memory superadmin. Env key is unchanged.

## Changes

- In `internal/middleware/admin_auth.go` `principalForDBKey`, delete the `OperatorRoleID == nil` branch that calls `SuperadminPrincipal(KindAPIKey)`.
- Null role on an admin key is 401 with the same body as unknown. Not 500. Not viewer coerce.
- `SuperadminPrincipal` remains for `KindEnvKey` only at the env-key compare in `AdminAuthMiddleware`.
- Roster: an admin key with an empty role name is a test failure. App keys are not on the roster.
- Invert `TestAdminAuth_NullRoleIsSuperadmin` in `internal/middleware/admin_auth_test.go`. Today it expects 200 and `admin_iam:write`. After this phase it is 401 and no principal. Keep env-key `SuperadminPrincipal` tests.

Do not add a config flag that restores fail-open.

## Data structures

No new types. `principalForDBKey` either returns a `Principal` from `RoleGrants` or aborts.

## Verification

Static: `go test -count=1 ./internal/middleware ./internal/operator ./internal/coreapp`.

Runtime: httptest nil-role admin key GET `/admin/x` → 401. Env key still `KindEnvKey` superadmin. Viewer DB key still 200 with no `tenants:write`.
