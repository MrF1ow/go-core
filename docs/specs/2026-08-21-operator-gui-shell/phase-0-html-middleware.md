# Phase 0: HTML middleware

[Overview](overview.md)

**Status:** landed on [PR #12](https://github.com/MrF1ow/go-core/pull/12) as `feat(middleware): add HTML RequireGUIPermission sibling`.

## Goal

An HTML sibling of JSON `RequireOperatorPermission` exists and is tested. No route calls it yet.

## What landed

- `internal/middleware/gui_permission.go`. `RequireGUIPermission`, `AbortGUIForbidden`, `AbortGUIInternal`, `sanitizeHTMXTarget`.
- `web/context_keys.go`. `GUIForbiddenHeader = "X-GUI-Forbidden"`, `GUIForbiddenValue = "1"`.
- Viewer deny is 403 HTML, never JSON. Fragment targets keep a sanitized id. Crafted `HX-Target` falls back to `page-content`. Missing principal is 500 HTML.
- JSON `RequireOperatorPermission` is unchanged.

## Still open

`AbortGUIForbidden` uses `c.String` with a `<div id="...">`. Phase 3 replaces that with templates. `requireGUI` does not exist in `internal/coreapp/app.go`. Inventory does not scan `guiAuth`.

Do not re-implement this phase. Do not merge #12 on this commit alone.
