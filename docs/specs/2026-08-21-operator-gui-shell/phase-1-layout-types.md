# Phase 1: Layout types

[Overview](overview.md)

## Goal

The layout root can carry a fail-closed `Can` and a computed nav. Nothing in `base.tmpl` changes yet.

## Changes

- Add `NavGroup` / `NavItem` on `web.TemplateData`, plus an unexported `can` func and a `Can(resource, action string) bool` method. Nil `can` returns false. A forgotten `page()` must paint an empty nav, not a full one.
- Add frozen `NavSpec` and `buildNav`. Filter by `can`. Drop groups whose item list is empty. Footer My Account and logout are not in the spec.
- `admin_iam` has no nav row.
- `web` does not import `operator`. `NavSpec` stores resource and action as strings. A test pins those strings against `operator.Catalog()` so the two lists cannot drift.
- `buildNav` lives in `web` so both `page()` and `AbortGUIForbidden` can call it without an `admin` ↔ `middleware` import cycle.

Do not range `NavGroups` in `base.tmpl` in this phase. Do not add `page(c)` yet.

## Data structures

```
TemplateData { NavGroups []NavGroup; can func(string, string) bool }
NavGroup { Heading string; Items []NavItem }
NavItem { Page, Path, Icon, Label string }
NavSpec { Heading, Page, Path, Icon, Label, Resource, Action string }
func (td TemplateData) Can(resource, action string) bool
func buildNav(basePath string, can func(string, string) bool) []NavGroup
```

Frozen rows match today's sidebar, in order. Dashboard has no heading. Then Management (tenants, applications, users, oauth, oidc-clients, session-groups), Security (sessions, ip-rules, roles, permissions, user-roles), Email (email-servers, email-templates, email-types), System (logs, api-keys, webhooks, monitoring, settings). Each row is `:read` of its catalog resource. Roles, permissions, and user-roles are `end_user_rbac`, not `admin_iam`.

## Verification

Static: `go test -count=1 ./web ./internal/operator`.

Runtime:

- Viewer `can` yields Dashboard, Users, Activity Logs, System Health. No Tenants. No empty Email heading. Security is absent. System still has logs and monitoring, not API Keys.
- Support `can` yields Dashboard, Users, Sessions, Activity Logs. No Tenants.
- Superadmin `can` yields every current sidebar row, still no `admin_iam` item.
- Nil `can` yields no groups.
- Catalog pin test fails if a `NavSpec` resource is not in `operator.Catalog()`.
