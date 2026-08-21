# Phase 5: Wire routes

[Overview](overview.md)

## Goal

Every `guiAuth` capability registration has `requireGUI`. Logout and My Account do not. A new unguarded `guiAuth.GET` fails CI.

## Changes

- Add `requireGUI(resource, action string)` in `internal/coreapp/app.go` as a one-line wrapper around `middleware.RequireGUIPermission`. The inventory AST needs a stable Ident, same as `requireOp`.
- Put `requireGUI` on each capability registration. Never `guiAuth.Use`. Follow [routes.md](routes.md) exactly. Nested exceptions are in that file so they are not rediscovered by prefix.
- Extend `internal/coreapp/require_op_inventory_test.go` (or a sibling in the same package) to scan `guiAuth` and identifiers assigned from `guiAuth.Group(...)`. Allowlist exact `/logout` and prefix `/my-account`. Do not allowlist by nav membership. Keep the existing JSON scan.
- Land the wrapper, the registrations, and the inventory in the same commit. Inventory goes red if the scan lands first.

This phase will touch `app.go` heavily. That is expected. Do not split by resource. A half-wired GUI is a false sense of safety.

IAM handler checks from phase 4 must already be in. Wiring `POST /api-keys` without the stamp gate leaves the mint hole open for any role that has `api_keys:write` (admin).

## Data structures

No new types. The inventory walks `go/ast` CallExpr args for Ident `requireGUI`, plus a string-literal path allowlist.

## Verification

Static: `go test -count=1 ./internal/coreapp`.

Runtime:

- Fixture source with `guiAuth.GET("/tenants", a.guiHandler.TenantPage)` fails the inventory.
- Fixture `guiAuth.GET("/logout", a.guiHandler.Logout)` passes.
- Fixture `guiAuth.GET("/my-account/2fa/status", ...)` passes.
- Fixture `guiAuth.GET("/users/:id/sessions", a.guiHandler.UserSessions)` without `requireGUI` fails.
- After wiring, `TestOperatorJSONRoutesRequirePermissionOnEachRegistration` still passes.
- Viewer cookie GET `/gui/tenants` is 403 with `X-GUI-Forbidden: 1`. Superadmin is 200. This can live in this phase or wait for phase 7. It must exist before merge.
