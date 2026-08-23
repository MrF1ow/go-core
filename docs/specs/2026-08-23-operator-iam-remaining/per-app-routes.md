# Per-app route map

Companion to [overview.md](overview.md). Frozen split for phases 7 and 8. Resource names are `operator.Res*` values.

A bound principal (`AppID != nil`) is denied on every platform resource, even when `Has` is true. App-scoped resources still need `Has`.

## Platform resources

These stay nil-AppID only.

| Resource | Why |
|----------|-----|
| `tenants` | Tenant CRUD is platform |
| `applications` | Creating or listing every app is platform. Bound operators do not manage the app row itself in v1 |
| `admin_iam` | Roster, custom roles, operator accounts |
| `settings` | Process-wide overrides |
| `monitoring` | Process health |
| `email` | SMTP and templates are not per-app in the GUI today |
| `oidc` | Client management spans apps in the admin GUI |
| `session_groups` | Cross-app SSO is tenant-wide |

## App-scoped resources

Bound operators may use these when `Has` allows, filtered to `p.AppID`.

| Resource | GUI area |
|----------|----------|
| `dashboard` | Counts for that app only in phase 8. Until then the existing `dashboard:read` cards stay platform and the guard in phase 7 **denies** dashboard for bound operators. Do not show mixed-tenant counts |
| `users` | Users with that `app_id` |
| `sessions` | Sessions for those users |
| `logs` | End-user `activity_logs` for that app |
| `api_keys` | App-type keys for that app. No admin keys |
| `oauth` | OAuth configs for that app |
| `ip_rules` | IP rules for that app |
| `webhooks` | Webhooks for that app |
| `end_user_rbac` | Roles and user_roles for that app |

Dashboard is listed twice on purpose. Phase 7 denies it for bound operators so they do not see global cards. Phase 8 may restore `/` and `/dashboard/stats` with app-filtered counts. If phase 8 skips dashboard, bound operators have no home page except Users. Prefer restoring a filtered dashboard in phase 8.

## Logout and my-account

Still session-only. No `requireGUI`. AppID does not apply.
