# Phase 8: Per-app lists

[Overview](overview.md)

PR: Per-app lists. Base is Per-app guard.

## Goal

When AppID is set, app-scoped pages only show that application and its users, keys, OAuth, IP rules, and webhooks.

## Changes

- Filter list and detail handlers named in [per-app-routes.md](per-app-routes.md) app-scoped table.
- Foreign app id in the URL is 404, not 403, so the operator cannot probe other apps.
- API key create as a bound operator forces that `app_id` and key type app. They cannot mint an admin key.
- httptest bound cookie list users from another app is empty. Direct GET of that user is 404.

Do not add a second login. Do not add tenant-level operators. One app per account is the v1 shape.

## Data structures

No new types. Handlers read `p.AppID` after the guard. Internal list functions take the app UUID they already have.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/coreapp`.

Runtime: two apps, users in each. Bound viewer on app A lists only A. Crafted `/gui/users/:id` for a B user is 404. Platform superadmin still sees both.
