# Operator IAM

**Date:** 2026-08-22
**Status:** shipped through [PR #50](https://github.com/MrF1ow/go-core/pull/50). Parked leftovers: [deferred](2026-09-03-per-app-admin-keys/deferred.md).

Operator IAM gates `/admin` JSON and `/gui`. It is a frozen `resource:action` catalog plus `Principal.Has`. It is not the app-user `roles` and `user_roles` tables, and it is not Session Groups. Those stay under `end_user_rbac`.

## What shipped

- JSON `requireOp` and GUI `requireGUI` on every capability route
- The sidebar and write buttons hide what `Can` denies
- API-key stamp via `ParseAssignedAdminRole`. Custom roles show on the roster role select and on API key create/edit. Account create always stamps viewer
- JSON and GUI roster, with CSV
- Evidence tables in migration `018`. Access logs, IAM events, CSV export, 365-day delete. IP and user agent on access logs (`020`)
- JSON and GUI account create, role change, and disable. Demote or disable of the last enabled superadmin is 409. Create 409 is a duplicate username
- Null `api_keys.operator_role_id` on an admin key is 401. The env key stays synthetic superadmin
- CSRF 403 is HTML and does not send `X-GUI-Forbidden`
- Custom operator roles. Non-system roles cannot grant `admin_iam`
- Admin keys must expire (`021`). Empty create is 400. Create form defaults to now plus 90 days
- One-way revoke and disable (`023`). No key delete. Disable cannot clear
- JSON `POST /admin/operator/keys` mints admin keys. Empty `app_id` is platform. Set `app_id` binds the key. Posted `key_type=app` still inserts `key_type=admin`. The GUI still mints admin keys with null `app_id`
- Per-app GUI operators (`022`). Bound JSON lists stamp `Principal.AppID` and filter like the GUI. Bound admin keys drop `api_keys_admin_app_id_null` (`024`). Superadmin keys stay platform. App delete is 409 while an admin key is bound

Dashboard `/gui/dashboard/stats` stays `dashboard:read`. Bound operators see counts for that app only.

## Per-app route map

A bound principal (`AppID != nil`) is denied on every platform resource, even when `Has` is true. App-scoped resources still need `Has`. Code: `internal/operator/platform.go`.

### Platform resources

These stay nil-AppID only.

| Resource | Why |
|----------|-----|
| `tenants` | Tenant CRUD is platform |
| `applications` | Creating or listing every app is platform. Bound operators do not manage the app row itself |
| `admin_iam` | Roster, custom roles, operator accounts |
| `settings` | Process-wide overrides |
| `monitoring` | Process health |
| `email` | SMTP and templates are not per-app in the GUI today |
| `oidc` | Client management spans apps in the admin GUI |
| `session_groups` | Cross-app SSO is tenant-wide |

### App-scoped resources

Bound operators may use these when `Has` allows, filtered to `p.AppID`.

| Resource | GUI area |
|----------|----------|
| `dashboard` | Counts for that app only |
| `users` | Users with that `app_id` |
| `sessions` | Sessions for those users |
| `logs` | End-user `activity_logs` for that app |
| `api_keys` | App-type keys for that app. No minting admin keys from a bound GUI session |
| `oauth` | OAuth configs for that app |
| `ip_rules` | IP rules for that app |
| `webhooks` | Webhooks for that app |
| `end_user_rbac` | Roles and user_roles for that app |

Logout and my-account stay session-only. No `requireGUI`. AppID does not apply.

## Still later

Parked items are [deferred](2026-09-03-per-app-admin-keys/deferred.md). `prevent_last_superadmin_cleared` raises when an update would leave zero enabled platform superadmins. SOC 2 Type I/II organizational evidence stays out of this repository.

## Do not build

- A `Grants()` lattice or a Redis grant store
- Cannot-grant-above-self. `admin_iam:write` is the stamp gate
- JSON mint of app-type keys on `POST /admin/operator/keys`. Bind owns `app_id`
- Duration preset buttons. Create already defaults to now plus 90 days
- GUI delete of operator accounts. Disable is the leaver path
- App-type API keys as operator principals
- The stale plan on [PR #7](https://github.com/MrF1ow/go-core/pull/7)

## Where it lives

`internal/operator` owns the catalog, stamp, last-superadmin check, and evidence cleanup. `internal/admin/operator_handler.go` owns JSON. `internal/admin/gui_iam.go` owns GUI IAM pages. `internal/coreapp/app.go` registers both.

Evidence retention and export are in [activity logging](../activity-logging.md). GUI pages are in [admin GUI](../admin-gui.md). JSON routes are in [API endpoints](../api-endpoints.md).
