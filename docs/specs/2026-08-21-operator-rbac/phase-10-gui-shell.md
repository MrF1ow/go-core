# Phase 10: GUI shell

[Overview](overview.md)

## Goal

The sidebar and `/gui/*` handlers use the same permission function as JSON. A support account sees Users, Sessions, Logs, Dashboard, My Account. They get 403 on Settings. A typed URL does not bypass the sidebar.

## Changes

- Register `RequireOperatorPermission` on `guiAuth` routes with the same resource map as JSON, plus GUI-only resources (`dashboard`, `session_groups`, `settings`, `api_keys`, `monitoring`, `admin_iam` pages).
- Logout, My Account, and static/login stay unrestricted for an authenticated session (My Account is self-service).
- `base.tmpl` wraps each nav link (except My Account) in a permission test. Pass a `Can` func in the template func map or a `map[string]bool` on `TemplateData`. Empty heading hides when no child is visible.
- HTMX list/detail/write routes for an entity use the same resource as the page. Support can `users:write` (toggle, unlock) and cannot `settings:write`.
- GUI for operator IAM: list accounts, assign role, create account, list custom roles, create custom role from the catalog. All `admin_iam`. Superadmin only in the seed.
- 403 page or HTMX alert. Do not redirect to dashboard as if the resource were missing.
- Tests: render sidebar for support (Tenants absent, Users present). Handler 403 for support on settings PUT. Superadmin sees IAM nav.
- **how** on `base.tmpl` and GUI route block before editing. **interrogate** if anyone proposes CSS hide instead of omitted markup.

## Data structures

Template gets `Can(resource, action) bool` closing over the principal. No second nav config file. **Laziness Protocol.**

## Verification

Static: `go test -count=1 ./web ./internal/admin ./internal/middleware`.

Runtime: `Instance("base" or a page that uses the layout)` with a support principal. Assert `data-page="tenants"` absent and `data-page="users"` present. httptest PUT settings → 403.

No browser driver in this environment. Flag control-ui as unavailable. Manual click-through is optional after merge.
