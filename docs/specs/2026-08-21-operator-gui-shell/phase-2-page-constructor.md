# Phase 2: page constructor

[Overview](overview.md)

## Goal

Every full page that includes `base.tmpl` is built with one constructor. Viewer bookmark of `/gui/users` gets an honest layout root. Fragments stay fragments.

## Changes

- Add `func (h *GUIHandler) page(c *gin.Context) web.TemplateData`. It stamps theme, username, admin id, CSRF, `can` from the request principal's `Has`, and `NavGroups` from `buildNav`. Missing principal leaves `can` nil.
- Replace every `gin.H` root that renders a `base.tmpl` page. Known offenders today: `UserPage`, `LogsPage`, `SessionsPage`, `ApiKeysPage`, `ApiKeyUsagePage`, `WebhookPage`, and their error paths. Several of those pass `AdminUser` while `base.tmpl` reads `AdminUsername`, so the footer name is already blank on a users bookmark.
- Existing `web.TemplateData{...}` literals in full-page handlers move onto `page(c)` plus `ActivePage` / `Data` / flash fields. Do not hand-copy username and CSRF next to the constructor.
- Login, 2FA verify, and other unauthenticated pages stay as they are. They have no sidebar.
- HTMX fragments (`*_list`, forms, confirms, dashboard stats) stay as their own data structs or `gin.H`. They do not include `base.tmpl`.

This is the migrate-and-delete wave for layout roots. Do not leave a dual path.

## Data structures

No new types. `page(c)` is the only producer of a `base.tmpl` `TemplateData`.

## Verification

Static: `gofmt -l` on `internal/admin`. `go test -count=1 ./internal/admin ./web`.

Runtime:

- `UserPage` through the renderer includes `AdminUsername` and `NavGroups`. A grep of `gui_handler.go` for `"AdminUser"` on layout roots is empty.
- Viewer principal `page(c)` `NavGroups` omits Tenants. Superadmin includes them.
- Login page still renders without a principal and without panicking on `Can`.
