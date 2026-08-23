# Operator IAM

**Date:** 2026-08-22
**Status:** shipped through [PR #30](https://github.com/MrF1ow/go-core/pull/30)

Operator IAM gates `/admin` JSON and `/gui`. It is a frozen `resource:action` catalog plus `Principal.Has`. It is not the app-user `roles` and `user_roles` tables, and it is not Session Groups. Those stay under `end_user_rbac`.

## What shipped

- JSON `requireOp` and GUI `requireGUI` on every capability route
- The sidebar and write buttons hide what `Can` denies
- API-key stamp via `ParseAssignedAdminRole`. Custom roles show on the roster role select and on API key create/edit. Account create always stamps viewer
- JSON and GUI roster, with CSV
- Evidence tables in migration `018`. Access logs, IAM events, CSV export, 365-day delete
- JSON and GUI account create, role change, and disable. Demote or disable of the last enabled superadmin is 409. Create 409 is a duplicate username
- Null `api_keys.operator_role_id` on an admin key is 401. The env key stays synthetic superadmin
- CSRF 403 is HTML and does not send `X-GUI-Forbidden`
- Custom operator roles. Non-system roles cannot grant `admin_iam`

Admin API keys are created in the GUI. JSON has `PUT /admin/operator/keys/:id/role` only.

## Still later

Product work is in [remaining](../2026-08-23-operator-iam-remaining/overview.md). SOC 2 Type I/II organizational evidence stays out of that directory too.

Dashboard `/gui/dashboard/stats` stays `dashboard:read`. That is a locked decision, not unfinished work.

## Do not build

- A `Grants()` lattice or a Redis grant store
- Cannot-grant-above-self. `admin_iam:write` is the stamp gate
- A JSON API-key create handler
- GUI delete of operator accounts. Disable is the leaver path
- App-type API keys as operator principals
- The stale plan on [PR #7](https://github.com/MrF1ow/go-core/pull/7)

## Where it lives

`internal/operator` owns the catalog, stamp, last-superadmin check, and evidence cleanup. `internal/admin/operator_handler.go` owns JSON. `internal/admin/gui_iam.go` owns GUI IAM pages. `internal/coreapp/app.go` registers both.

Evidence retention and export are in [activity logging](../activity-logging.md). GUI pages are in [admin GUI](../admin-gui.md). JSON routes are in [API endpoints](../api-endpoints.md).
