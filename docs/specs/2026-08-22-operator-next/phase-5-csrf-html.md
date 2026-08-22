# Phase 5: CSRF HTML

[Overview](overview.md)

PR: CSRF. Independent of IAM.

## Goal

A CSRF miss on the GUI is an HTML 403. It is not JSON. HTMX does not swap it as an IAM deny.

## Changes

- `CSRFMiddleware` stops calling `AbortWithStatusJSON` for missing and invalid tokens.
- Write HTML via the same `guiLayoutData` ingredients `AbortGUIForbidden` uses. Dedicated templates `csrf_forbidden` and `csrf_forbidden_fragment`. Do not name them `"error"`. Do not reuse `forbidden.tmpl`. CSRF is not missing a grant.
- Do not set `X-GUI-Forbidden`. `TestCSRFForbiddenDoesNotSendGUIHeader` stays. Change it so the body is HTML, not `application/json`.
- Typed URL POST gets the page template. HTMX POST without the header stays unswapped. Do not turn on global 4xx swap. Do not `HX-Redirect`.
- Settings env-lock, if it still JSON-aborts, stays its own 403. Do not reuse the CSRF template for it.

JSON `/admin` is unchanged. `RequireOperatorPermission` stays JSON.

## Data structures

No new types. Reuse `guiLayoutData` in `internal/middleware/gui_permission.go`.

## Verification

Static: `go test -count=1 ./internal/middleware ./web`.

Runtime: POST `/gui/tenants` with a session and no CSRF token → 403 HTML, no `X-GUI-Forbidden`, body is not the JSON `{"error":"CSRF token missing"}`. IAM viewer GET `/gui/tenants` still sends the header. JSON `RequireOperatorPermission` deny is still JSON.
