# Phase 3: Forbidden templates

[Overview](overview.md)

## Goal

A GUI deny is a real page or a real fragment, and HTMX will swap it. CSRF JSON 403 and settings env-lock stay unswapped.

## Changes

- Add `web/templates/pages/forbidden.tmpl`. It goes through `base` and defines `#page-content`. Typed URL and `HX-Target` of `#page-content` use it so `hx-select` still has a node and the sidebar still omits via `Can`.
- Add `web/templates/partials/forbidden_fragment.tmpl`. Any other HTMX target uses it. The fragment has no outer id, because `hx-swap="innerHTML"` would nest a second element with that id inside the real container. Crafted `HX-Target` that fails `[A-Za-z][A-Za-z0-9_-]{0,127}` is treated as a page deny.
- `AbortGUIForbidden` switches from `c.String` to `c.HTML`. It still sets `X-GUI-Forbidden: 1`. It still never JSON-aborts. Build the page-shaped body with the same username / CSRF / `Can` / `NavGroups` ingredients `page(c)` uses, from the gin context, so a typed-URL 403 does not paint a full sidebar.
- Do not name either template `"error"`.
- Add a `htmx:beforeSwap` listener in `base.tmpl` that sets `shouldSwap` only when the response header is `X-GUI-Forbidden: 1`. Not global 4xx swap. Not `HX-Redirect`.
- CSRF middleware and settings env-lock must not send that header. Add a test that a CSRF 403 does not.

`AbortGUIInternal` can stay `c.String` 500. It is a bug, not a permission deny.

## Data structures

No new types. Header constants already exist on `web`.

## Verification

Static: `go test -count=1 ./internal/middleware ./web`.

Runtime, extending the phase 0 tests:

- Viewer typed GET deny contains `#page-content` and does not contain the JSON error shape.
- HTMX deny with target `#page-content` is the page template, header present.
- HTMX deny with target `#tenant-table` is a fragment that does not repeat `id="tenant-table"`.
- Crafted target `"><script>` falls back to the page deny.
- A fixture CSRF 403 is not swapped by the listener contract (header absent).
- JSON `RequireOperatorPermission` is still JSON.
