# Operator GUI shell

**Date:** 2026-08-21
**Status:** implementation in progress on [PR #12](https://github.com/MrF1ow/go-core/pull/12)
**Depends on:** [PR #10](https://github.com/MrF1ow/go-core/pull/10) (merged). JSON `requireOp` and GUI principal attach are on main.
**Parent:** [PR #7](https://github.com/MrF1ow/go-core/pull/7) old phase 10.

This directory is the rest of the work that makes #12 mergeable. A helper nobody calls is not a merge. Do not start a second implementation PR.

## Context

JSON `/admin` already enforces the frozen `resource:action` catalog. GUI cookies carry `KindGUIAccount` and still never call `Has`. A valid `admin_session` is all-powerful. Any logged-in cookie can mint a superadmin JSON key from `/gui/api-keys`.

[PR #12](https://github.com/MrF1ow/go-core/pull/12) already landed `RequireGUIPermission`, `AbortGUIForbidden`, and `X-GUI-Forbidden: 1`. Deny bodies are still `c.String` HTML. No `guiAuth` registration calls the middleware. `base.tmpl` still paints every nav link. `resolveAdminOperatorRoleID` still stamps any system role the poster asks for.

The grant model is not the hard part. The HTTP and HTML adapter is. `guiAuth` also hosts logout and `/my-account/*`, so a group-level `Use` is wrong. Layout pages mix `TemplateData` and `gin.H`, so a `.Can` method has no honest home. HTMX 2.0.4 does not swap 4xx, so an HTML 403 that is merely "not JSON" still leaves stale `#page-content`. Nested `/users/:id/sessions` and API-key role stamping are compound operations a single middleware line cannot express.

## Scope

Included, all on #12, as later commits:

- `page(c)` as the only `base.tmpl` root. Unify the `gin.H` layout pages.
- `TemplateData.Can` over an unexported func. Computed `NavGroups`. Sidebar omit.
- `forbidden.tmpl` / `forbidden_fragment.tmpl` plus the `X-GUI-Forbidden` HTMX listener.
- `ParseAssignedAdminRole` so `api_keys:write` cannot stamp superadmin, admin, or support without `admin_iam:write`.
- `requireGUI` on every `guiAuth` capability registration. Inventory scan of `guiAuth` and `Group()` derivatives.
- Write CTA omit via `Can` on the list and form templates this slice touches.
- Role httptests that are the merge bar.

Excluded, still the parent plan or later:

- Roster UI, IAM history, access log
- JSON operator-account CRUD, last-superadmin, custom roles UI, `disabled_at`
- CSRF JSON-to-HTML
- Flipping null `api_keys.operator_role_id` off fail-open superadmin
- Dashboard stats vs custom roles (stats stay `dashboard:read`)

## Constraints

- Same catalog. Same `Principal.Has`. No Redis grant store. No `Grants()` lattice.
- Never `guiAuth.Use(requireGUI)`. Logout exact `/logout` and prefix `/my-account` stay session-only.
- JSON `RequireOperatorPermission` stays JSON. GUI deny is HTML 403 plus `X-GUI-Forbidden: 1`.
- `web` does not import `operator`.
- Do not name a template `"error"`. That template does not exist.
- Form GETs that start a mutation are `:write`. Page, list, form-cancel, export, and detail are `:read`.
- Nested exceptions, not prefix inheritance. See [shape.md](shape.md) and [routes.md](routes.md).
- Inventory is the lever. Do not allowlist by "is it in the nav." Handler-local IAM is a dedicated test, not an AST property.
- No control-ui in this environment. Verify with `go test` and httptest.
- Do not merge #12 between the wiring commit and the sidebar/CTA commit. Naked 403s on every Create Tenant button are not mergeable.

## Alternatives

Already decided. Base is arena candidate 2. Grafts from candidate 1. See [shape.md](shape.md).

Rejected and still rejected:

- JSON/HTML flag on `RequireOperatorPermission`
- `guiAuth.Use(requireGUI)`
- `HX-Redirect` every deny
- Global HTMX 4xx swap
- CSS-hide nav
- Prefix heuristic for nested sessions
- `Grants()` lattice
- RouteSpec DSL that replaces gin

Wiring every `guiAuth` route in one commit looks oversized against the two-to-three-file phase rule. Inventory goes red if the scan lands before the registrations. That is the reason they stay one commit. **Encode lessons in structure** plus **outcome-oriented execution**.

## Merge bar

#12 is mergeable when all of these pass on a cookie session, not only on a unit principal:

- Viewer GET `/gui/tenants` is 403 HTML with `X-GUI-Forbidden: 1`. Superadmin is 200.
- Viewer sidebar includes Dashboard, Users, Activity Logs, System Health. It omits Tenants. Empty Email heading is gone.
- Viewer cannot see Create Tenant, user Import, or the API-key operator role select.
- Support GET `/gui/users/:id/sessions` is 200. Viewer crafted GET is fragment 403. Viewer user detail still loads.
- Admin cookie POST `/gui/api-keys` with `operator_role_id=superadmin` is 403, no row. Omit the field and the key is viewer.
- Logout and `/my-account` work with no `requireGUI`.
- A new `guiAuth.GET` without `requireGUI` fails the inventory test.

Do not merge after wiring only.

## Applicable skills

- `go-core` hub, then `admin-gui.md`, `route-map.md`, `security.md`
- `how` over `GUIAuthMiddleware`, `gui_handler.go` layout roots, and the inventory AST before editing them
- `interrogate` before marking #12 ready
- `unslop` on docs and PR copy
- `/deslop` before each commit

## Phases

Commits on `cursor/operator-gui-shell-impl-a895`. One PR.

0. [HTML middleware](phase-0-html-middleware.md) (landed on #12)
1. [Layout types](phase-1-layout-types.md)
2. [page constructor](phase-2-page-constructor.md)
3. [Forbidden templates](phase-3-forbidden-templates.md)
4. [IAM stamp](phase-4-iam-stamp.md)
5. [Wire routes](phase-5-wire-routes.md)
6. [Sidebar and write CTAs](phase-6-sidebar-ctas.md)
7. [Role httptests](phase-7-role-httptests.md)

Companion: [shape.md](shape.md), [routes.md](routes.md), [testing.md](testing.md).

## Verification

```
gofmt -l on touched Go files (empty)
go test -count=1 ./internal/middleware ./internal/admin ./internal/operator ./internal/coreapp ./web
```

Full merge bar is phase 7. Each earlier phase has its own check. See [testing.md](testing.md).

## Implementation guidance

- Run **how** on `internal/coreapp/app.go` `guiAuth` registrations and `web/templates/layouts/base.tmpl` before the wiring and sidebar commits.
- **Foundational thinking.** Types and `page(c)` before route guards. Inventory before relying on greps.
- **Boundary discipline.** `RequireGUIPermission` stays HTML. `ParseAssignedAdminRole` stays pure in `internal/operator`. Handlers map `ErrIAMAssignmentDenied` to `AbortGUIForbidden`.
- **Type system discipline.** Nil `Can` denies. Unexported `can` so `gin.H` cannot grow a fake one. `web` does not import `operator`.
- **Model the domain.** Frozen `NavSpec` plus `buildNav`. Do not sprinkle `{{if .Can}}` on every sidebar link.
- **Encode lessons in structure.** Extend the existing AST inventory. Do not write "remember requireGUI" in a comment.
- **Build the lever.** The inventory test is the lever for ~100 registrations. The frozen map in [routes.md](routes.md) is what the implementer applies, not a second handwritten pass.
- **Migrate callers then delete legacy APIs.** `gin.H` layout roots go away in phase 2. Do not leave `AdminUser` keys next to `page(c)`.
- **Experience first.** Sidebar omit and write CTA omit land together. First paint is the only sidebar paint.
- **Sequence work into verifiable units.** The commit order above is the review story.
- **Prove it works.** httptest through `RegisterRoutes` with a viewer cookie, not only `Principal.Has` in a unit test.
- **Laziness protocol.** Reuse `p.Has`. Do not add `Grants()`. Do not wrap `HasScope` for `/admin`.
- **Never block on the human** for reversible internals. Do not change seeded role names or catalog rows.
- **Interrogate** before marking #12 ready for review.
- Cursor babysit after the PR leaves draft.

## Consumer and maintainer

Consumer. GUI login still works after #10. Extra `cmd/setup` accounts are viewer. Apply migration 017 before new GUI accounts. After #12 merges, a viewer cookie is actually restricted. An admin cookie can still use the GUI and mint viewer JSON keys. It cannot mint superadmin.

Maintainer. Next after #12 is roster, IAM history, and access log from PR #7. Custom roles UI stays out. Null API-key role fail-open stays until a dedicated cutover.
