# Operator GUI shell

**Date:** 2026-08-21
**Status:** design, not implemented
**Depends on:** [PR #10](https://github.com/MrF1ow/go-core/pull/10) (GUI principal attach). JSON `requireOp` is already on main.
**Parent:** [PR #7](https://github.com/MrF1ow/go-core/pull/7) old phase 10.

## Problem

JSON `/admin` already enforces the frozen `resource:action` catalog. GUI cookies already carry `KindGUIAccount` once #10 lands, then never call `Has`. A valid `admin_session` is still all-powerful.

The grant model is not the hard part. The HTTP and HTML adapter is. `RequireOperatorPermission` always JSON-aborts. `guiAuth` also hosts logout and `/my-account/*`, so a group-level `Use(requireOp)` is wrong. Layout pages mix `TemplateData` and `gin.H`, so a `.Can` method has no honest home. HTMX 2.0.4 does not swap 4xx, so an HTML 403 that is merely "not JSON" still leaves stale `#page-content`. Nested `/users/:id/sessions` and API-key role stamping are compound operations a single `requireOp` line cannot express.

## Usage

Route authors keep the JSON shape. Attach stays on `GUIAuthMiddleware`. Enforcement is `requireGUI(resource, action)` on each capability registration. Logout and `/my-account/*` omit it.

```go
guiAuth.GET("/logout", a.guiHandler.Logout)
guiAuth.GET("/my-account", a.guiHandler.MyAccountPage)
guiAuth.GET("/tenants", requireGUI(operator.ResTenants, operator.ActionRead), a.guiHandler.TenantPage)
guiAuth.GET("/tenants/new", requireGUI(operator.ResTenants, operator.ActionWrite), a.guiHandler.TenantCreateForm)
guiAuth.GET("/users/:id/sessions", requireGUI(operator.ResSessions, operator.ActionRead), a.guiHandler.UserSessions)
guiAuth.POST("/api-keys", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyCreate)
```

Every page that includes `base.tmpl` is built with one constructor. `gin.H` layout roots go away.

```go
func (h *GUIHandler) UserPage(c *gin.Context) {
    data := h.page(c)
    data.ActivePage = "users"
    data.Data = apps
    c.HTML(http.StatusOK, "users", data)
}
```

API-key role assignment is not a second `requireGUI`. The page is `api_keys:write`. Stamping superadmin, admin, or support is `ParseAssignedAdminRole`. Empty post stamps viewer with no IAM. A posted non-viewer role without `admin_iam:write` is HTML 403, not a silent coerce to viewer.

## Shape

Same catalog. Same `Principal.Has`. No Redis grant store. No `Grants()` lattice.

**Three ports, one shell.**

1. `RequireGUIPermission` is the HTML sibling of `RequireOperatorPermission`. Missing principal is 500 HTML. Deny is `AbortGUIForbidden`. Never `AbortWithStatusJSON`. Never a path-prefix switch inside the JSON middleware. `AdminBasePath` is configurable. JSON lives at `/admin` and `/metrics`.
2. `page(c)` is the only legal `base.tmpl` root. It stamps username, CSRF, theme, `Can`, and computed `NavGroups`. Nil `Can` denies, so a forgotten `page()` paints an empty nav, not a full one. `web` does not import `operator`. `Can` is a method over an unexported func filled from `p.Has`.
3. `ParseAssignedAdminRole` / `AssignableSystemRoles` live in `internal/operator`. They own "who may stamp which system role on an admin API key." Handlers map `ErrIAMAssignmentDenied` to the same `AbortGUIForbidden`.

**Deny protocol.** HTTP 403, header `X-GUI-Forbidden: 1`, HTML body. A `htmx:beforeSwap` listener swaps only those responses. Not global 4xx swap (that would start swapping CSRF JSON 403 and settings env-lock fragments). Not `HX-Redirect` on fragments (that would blow away user detail for a nested widget).

Typed URL or `HX-Target` of `#page-content` renders `forbidden.tmpl` through `base`, so `#page-content` exists for `hx-select` and the sidebar still omits via `Can`. Any other HTMX target renders `forbidden_fragment.tmpl`. Crafted `HX-Target` is sanitized to `[A-Za-z][A-Za-z0-9_-]{0,127}` or treated as a page deny.

**Nav.** `buildNav` filters a frozen `NavSpec` in Go and drops empty groups. The template ranges `NavGroups`. It does not sprinkle `{{if .Can}}` per link. Footer My Account and logout are not in `NavGroups`. First paint is the only sidebar paint. Route `Has` still applies on the next request.

**Explicit pairs.** Form GETs that start a mutation (`/new`, `/:id/edit`, delete confirm, import modal) are `:write`. Page, list, form-cancel, export, detail are `:read`. Read-only catalog rows stay read-only.

Nested exceptions, not prefix inheritance:

- `GET /users/:id/sessions` is `sessions:read`. Viewer gets user detail without the widget. Support gets rows. Crafted GET is a fragment 403.
- `GET /dashboard/activity` is `logs:read`. Stats stay `dashboard:read`.
- `POST /ip-rules/check` is `ip_rules:write`, same as JSON.
- End-user `/roles` is `end_user_rbac`, not `admin_iam`.

**Inventory.** Scan `guiAuth` and groups derived from it for ident `requireGUI`. Allowlist exact `/logout` and prefix `/my-account`. Do not allowlist by "is it in the nav." Handler-local IAM is a dedicated test, not an AST property.

## Synthesis decision

Two grok 4.6 high arena candidates converged on `requireGUI` plus a single layout constructor, `sessions:read` for nested user sessions, and `admin_iam:write` as a boolean stamp gate.

**Base is candidate 2.** The `X-GUI-Forbidden` listener is a deeper HTMX adapter than enabling every 403 swap. `ParseAssignedAdminRole` belongs in `operator`, not a GUI util. Unexported `Can` makes `gin.H` layout roots dishonest. Mutating-form GETs as `:write` hides `/users/import/modal` from viewers. `POST /ip-rules/check` stays write, matching JSON.

**Grafted from candidate 1.** Computed `NavGroups` so empty headings cannot leak from a forgotten template `or`. Inventory tracks `Group()` derivatives so a future subgroup cannot leave the scan. `HX-Target` sanitization so a crafted id cannot land in HTML.

**Rejected from both.** Reusing `RequireOperatorPermission` with a JSON/HTML flag. `guiAuth.Use(requireGUI)`. `HX-Redirect` every deny. CSS-hide nav. Prefix heuristic for nested sessions. A `Grants()` lattice. A RouteSpec DSL that replaces gin registrations.

## Tradeoffs accepted

- We accept a second middleware constructor in exchange for never teaching the JSON one about HTMX.
- We accept a 12-line `htmx:beforeSwap` listener in exchange for not swapping CSRF or env-lock 403s.
- We accept `admin_iam:write` as a boolean gate, not a cannot-grant-above-self lattice, in exchange for leaving `perms` unexported.
- We accept fail-closed on API-key edit of a non-viewer role without IAM in exchange for one parse function.
- We accept first-paint-only sidebar in exchange for not re-rendering nav on every HTMX swap.
- We accept CSRF remaining JSON 403 in this slice. Permission 403 must not copy it.
- We accept unifying full pages onto `TemplateData` as mandatory. Viewer bookmark of `/gui/users` is a `gin.H` first paint today.

## Out of this slice

Roster UI, IAM history, access log, JSON operator-account CRUD, last-superadmin, custom roles UI, `disabled_at`, CSRF JSON-to-HTML.

## Next implementation step

Add `RequireGUIPermission` and `AbortGUIForbidden` with tests that a viewer cookie gets HTML 403 (typed URL, `#page-content`, fragment) and never JSON. Then wire `requireGUI` on `/tenants` as the first registration pair. Stack on #10. Do not land on #10 itself.

## Verification

```
go test -count=1 ./internal/middleware ./internal/admin ./internal/operator ./internal/coreapp ./web
```

Viewer GET `/gui/tenants` is 403 HTML with `X-GUI-Forbidden: 1`. Superadmin is 200. Sidebar for support omits Tenants and includes Users. Admin cookie posting `operator_role_id=superadmin` on `/gui/api-keys` is 403, no row. Omit the field and the key is viewer.
