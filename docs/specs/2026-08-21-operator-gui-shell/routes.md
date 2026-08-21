# GUI requireGUI map

[Overview](overview.md)

Frozen pairs for phase 5. Method plus path as registered on `guiAuth` in `internal/coreapp/app.go`. Resource names are `operator.Res*` values. Action is `read` or `write`.

Rule. Form GETs that start a mutation (`/new`, `/:id/edit`, delete confirm, import modal, revoke confirm) are `:write`. Page, list, form-cancel, export, and detail are `:read`. Mutations are `:write`. Read-only catalog rows stay read-only.

## Session-only (no requireGUI)

| Method | Path |
|--------|------|
| GET | `/logout` |
| GET | `/my-account` |
| POST | `/my-account/email` |
| POST | `/my-account/password` |
| POST | `/my-account/2fa/generate` |
| POST | `/my-account/2fa/verify-totp` |
| POST | `/my-account/2fa/enable-email` |
| POST | `/my-account/2fa/disable` |
| GET | `/my-account/2fa/status` |
| POST | `/my-account/2fa/regenerate-codes` |
| GET | `/my-account/passkeys/status` |
| POST | `/my-account/passkeys/register/begin` |
| POST | `/my-account/passkeys/register/finish` |
| DELETE | `/my-account/passkeys/:id` |
| POST | `/my-account/passkeys/:id/rename` |
| GET | `/my-account/magic-link/status` |
| POST | `/my-account/magic-link/toggle` |
| GET | `/my-account/backup-email/status` |
| POST | `/my-account/backup-email` |
| DELETE | `/my-account/backup-email` |
| GET | `/my-account/trusted-devices` |
| DELETE | `/my-account/trusted-devices/:device_id` |

## Nested exceptions (do not inherit from the parent path)

| Method | Path | Pair |
|--------|------|------|
| GET | `/dashboard/activity` | `logs:read` |
| GET | `/users/:id/sessions` | `sessions:read` |
| POST | `/ip-rules/check` | `ip_rules:write` |

Dashboard stats stay `dashboard:read`. End-user `/roles`, `/permissions`, `/user-roles` are `end_user_rbac`, not `admin_iam`.

## Dashboard

| Method | Path | Pair |
|--------|------|------|
| GET | `/` | `dashboard:read` |
| GET | `/dashboard/stats` | `dashboard:read` |
| GET | `/dashboard/activity` | `logs:read` |

## Tenants (`tenants`)

read: `GET /tenants`, `GET /tenants/list`, `GET /tenants/form-cancel`

write: `GET /tenants/new`, `POST /tenants`, `GET /tenants/:id/edit`, `PUT /tenants/:id`, `GET /tenants/:id/delete`, `DELETE /tenants/:id`

## Applications (`applications`)

read: `GET /applications`, `GET /applications/list`, `GET /applications/form-cancel`

write: `GET /applications/new`, `POST /applications`, `GET /applications/:id/edit`, `PUT /applications/:id`, `GET /applications/:id/delete`, `DELETE /applications/:id`

## OAuth (`oauth`)

read: `GET /oauth`, `GET /oauth/list`, `GET /oauth/form-cancel`

write: `GET /oauth/new`, `POST /oauth`, `GET /oauth/:id/edit`, `PUT /oauth/:id`, `GET /oauth/:id/delete`, `DELETE /oauth/:id`, `PUT /oauth/:id/toggle`

## Users (`users`) except nested sessions

read: `GET /users`, `GET /users/list`, `GET /users/export`, `GET /users/:id`

write: `GET /users/import/modal`, `POST /users/import`, `PUT /users/:id/toggle`, `PUT /users/:id/unlock`, `GET /users/social-accounts/:id/unlink`, `DELETE /users/social-accounts/:id`, `GET /users/passkeys/:id/delete`, `DELETE /users/passkeys/:id`, `DELETE /users/:id/trusted-devices/:device_id`, `DELETE /users/:id/trusted-devices`

## Logs (`logs`)

read: `GET /logs`, `GET /logs/list`, `GET /logs/export`, `GET /logs/:id`

## API keys (`api_keys`)

read: `GET /api-keys`, `GET /api-keys/list`, `GET /api-keys/form-cancel`, `GET /api-keys/:id/usage`

write: `GET /api-keys/new`, `POST /api-keys`, `GET /api-keys/:id/edit`, `PUT /api-keys/:id`, `GET /api-keys/:id/revoke`, `PUT /api-keys/:id/revoke`, `GET /api-keys/:id/delete`, `DELETE /api-keys/:id`

Stamp gate on POST and PUT is phase 4, not a second `requireGUI`.

## Settings (`settings`)

read: `GET /settings`, `GET /settings/info`, `GET /settings/section/:category`

write: `PUT /settings/:key`, `DELETE /settings/:key`

## Monitoring (`monitoring`)

read: `GET /monitoring`, `GET /monitoring/health`, `GET /monitoring/metrics`

## Email (`email`)

Servers. read: page, list, form-cancel. write: new, POST, edit, PUT, delete confirm, DELETE, POST test.

Templates. read: page, list, form-cancel. write: new, POST, edit, PUT, delete confirm, DELETE, POST preview, POST editor-window, GET reset confirm, POST reset.

Variables. read: `GET /email-variables`.

Types. read: page, list, form-cancel. write: new, POST, edit, PUT, delete confirm, DELETE.

## End-user RBAC (`end_user_rbac`)

Roles. read: page, list, form-cancel, `GET /roles/:id/permissions`. write: new, POST, edit, PUT, delete confirm, DELETE, `PUT /roles/:id/permissions`.

Permissions. read: page, list, form-cancel. write: new, POST.

User-roles. read: page, list, form-cancel, `GET /user-roles/roles-for-app`, `GET /user-roles/search-users`. write: new, POST, PUT, GET revoke confirm, DELETE.

## Sessions (`sessions`)

read: `GET /sessions`, `GET /sessions/list`, `GET /sessions/:app_id/:session_id/detail`, `GET /users/:id/sessions`

write: `DELETE /sessions/:app_id/:session_id`, `DELETE /sessions/revoke-all-user`

## IP rules (`ip_rules`)

read: page, list, form-cancel

write: new, POST, edit, PUT, delete confirm, DELETE, `POST /ip-rules/check`

## Webhooks (`webhooks`)

read: page, list, form-cancel, `GET /webhooks/:id/deliveries`

write: new, POST, delete confirm, DELETE, PUT toggle

## OIDC (`oidc`)

read: page, list, form-cancel

write: new, POST, edit, PUT, delete confirm, DELETE, POST rotate-secret

## Session groups (`session_groups`)

read: page, list, form-cancel, `GET /session-groups/:id/apps`

write: new, POST, edit, PUT, delete confirm, DELETE, POST apps, DELETE app
