# Phase 6: Per-app session

[Overview](overview.md)

PR: Per-app session. Base is Per-app shape.

## Goal

GUI login attaches `Principal.AppID` from the account row. JSON admin keys stay nil AppID. Roster shows the app when set.

## Changes

- `GUIAuthMiddleware` loads `app_id` onto the principal.
- JSON `principalForDBKey` leaves AppID nil.
- Roster entry grows optional `app_id` / `app_name`. Env key and admin keys omit it.
- GUI account create and role forms gain an optional app select for IAM writers. Empty is platform. Superadmin posts with an app are 400.
- httptest a bound viewer cookie has AppID set.

Do not deny Tenants yet. Phase 7 owns the guard.

## Data structures

`RosterEntry` gains `AppID *uuid.UUID` and `AppName string`. JSON omitempty.

## Verification

Static: `go test -count=1 ./internal/middleware ./internal/admin ./internal/coreapp`.

Runtime: create a viewer bound to an app as superadmin. That account's GUI cookie principal has AppID. Roster row shows the app name. Superadmin create with an app plus superadmin role is 400.
