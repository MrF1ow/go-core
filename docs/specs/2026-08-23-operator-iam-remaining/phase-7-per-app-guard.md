# Phase 7: Per-app guard

[Overview](overview.md)

PR: Per-app guard. Base is Per-app session.

## Goal

A bound GUI operator cannot use platform routes. `Has` still decides the grant. AppID decides the scope.

## Changes

- Add `operator.PlatformResource(resource string) bool` from the frozen map in [per-app-routes.md](per-app-routes.md).
- `RequireGUIPermission` and `RequireOperatorPermission` deny when `p.AppID != nil` and the resource is platform. JSON 403. GUI HTML 403 plus `X-GUI-Forbidden: 1`.
- Access log still writes on those denies.
- Sidebar omit already hides what `Can` denies. Platform rows must also hide when AppID is set, even if the role has `tenants:read`.
- httptest a bound admin cookie GET `/gui/tenants` is 403. GET `/gui/users` is 200 when the role has `users:read`. Bound JSON key does not exist. Bound is GUI-only.

Admin API keys remain platform. Do not add AppID to them.

## Data structures

`func PlatformResource(resource string) bool`. Closed set. Test against `Catalog()` so an unknown resource cannot silently become app-scoped.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/middleware ./internal/coreapp ./web`.

Runtime: bound admin cookie GET `/gui/tenants` 403 fragment. GET `/gui/operator` 403. GET `/gui/users` 200. Platform superadmin cookie still 200 on Tenants.
