# Operator RBAC

**Date:** 2026-08-21
**Status:** plan, not implemented
**Back-link:** this directory is the plan

## Context

GUI operators and admin JSON callers are all-powerful today. `admin_accounts` have no role. `X-Admin-API-Key` authenticates and then every `/admin/*` handler runs. DB keys store a `scopes` string. Middleware parses it. No handler calls `HasScope`. A support-shaped key can still create tenants.

App-user RBAC (`admin` / `member` / `viewer` on `roles` / `user_roles`) is a different domain. Those roles are per `AppID` and gate the JWT API for people in the consumer product. They must not be reused for GUI operators or admin keys.

We need operator identity, least privilege, one permission check on every admin door, and a roster plus IAM history so “who had what” is queryable. That is the technical half of SOC 2 Security CC6. It is not a SOC 2 report.

## Scope

Included:

- New operator catalog (permissions, roles, grants). Not the app-user tables.
- Seeded roles: `superadmin`, `admin`, `support`, `viewer`.
- One principal on the request. Env key, DB admin key, or GUI account.
- `RequireOperatorPermission` on JSON `/admin/*`, `GET /metrics`, and admin OIDC JSON. Same function later on `/gui/*`.
- Env key runs the check with an in-memory full grant (synthetic superadmin). No skip.
- New DB admin keys and new GUI accounts default to `viewer`.
- Existing DB admin keys with blank `scopes` backfill to `superadmin` so current scripts do not lock out.
- Admin keys may expire. Default is forever (`api_keys.expires_at` null). An operator can set a timestamp. After that instant, `FindActiveKeyByHash` treats the key as unknown (401) before any permission check. Keep the existing column. Do not add a second TTL table. Edit must be able to set or clear expiry the same as create.
- `cmd/setup` first account is `superadmin`. Extra accounts created through setup are `viewer`.
- Last `superadmin` GUI account cannot be demoted or deleted.
- `My Account` and logout always allowed for a logged-in GUI session.
- Roster, IAM assignment history, and operator access log (allow and deny).
- Custom operator roles (`is_system=false`) that superadmin can create from the catalog.

Excluded (follow-up interactions, not this plan):

- Per-app or per-tenant GUI operators.
- App-type keys (`X-App-API-Key` on `/app/:id/*`) using this catalog.
- Binding a DB key to an `admin_account` (key remains its own principal).
- Replacing the static env key. It stays break-glass.
- Claiming SOC 2 Type I / Type II. Those are organizational (access reviews, retention policy, named keys for daily use).
- End-user JWT `admin` / `member` / `viewer` changes.

## Constraints

- Next migration is `016_`. Update `internal/schema.sql` and run `sqlc generate`. No `IF NOT EXISTS` on new tables. Seed with `ON CONFLICT DO NOTHING`.
- Do not import `internal/rbac` into operator IAM. Separate names: `operator_roles`, not `roles`.
- Public module API stays `app.New` / `RegisterRoutes` / `AuthMiddleware`. Do not export `RequireOperatorPermission` in v1 unless a test in this repo needs it through `internal/`.
- JSON 401 for bad or missing key. JSON 403 for authenticated principal missing a permission. GUI 403 HTML, not a silent redirect to dashboard.
- Existing `HasScope` stays for possible app-key follow-up. Admin JSON must not depend on `api_keys.scopes` after the cutover.
- Keep `api_keys.expires_at`. Null is forever. A past timestamp is dead, same as revoked, not a 403. Env key has no expiry.
- `activity_logs` stays end-user (`app_id` + `user_id`). Operator evidence is new tables.
- No control-ui / browser skill in this environment. JSON phases verify with `go test` and httptest. GUI phases verify with template and handler tests. Flag runtime-in-browser as unavailable.

## Alternatives

**A. Enforce the existing `scopes` string.** Smallest diff. Two vocabularies once GUI accounts exist. Form copy already disagrees with `HasScope` (blank = unrestricted vs deny). Rejected.

**B. Reuse app-user `roles`.** One RBAC service. Roles are `AppID`-scoped. Operators are global. Rejected. **Model the Domain.**

**C. Operator catalog + principal + one check (chosen).** Keys and GUI accounts share `operator_role_id`. Env key is a synthetic full grant. Same middleware. **Foundational Thinking** (principal first). **Redesign from First Principles** (authz as if it had always been on the route).

## Applicable skills

- `go-core` hub, then `admin-gui.md`, `route-map.md`, `data-model.md`, `security.md`
- `how` over middleware and `internal/coreapp/app.go` before editing route registration
- `interrogate` before shipping the JSON cutover (phase 3) and the GUI cutover (phase 9)
- `unslop` on docs and PR copy
- `/deslop` before each commit
- `show-me-your-work` for the cutover decisions (backfill, last superadmin)

## Phases

1. [Catalog schema](phase-1-catalog-schema.md)
2. [Seed roles](phase-2-seed-roles.md)
3. [Principal and check](phase-3-principal-and-check.md)
4. [JSON route enforcement](phase-4-json-enforcement.md)
5. [Key lifecycle](phase-5-key-lifecycle.md)
6. [Roster](phase-6-roster.md)
7. [IAM history](phase-7-iam-history.md)
8. [Access log](phase-8-access-log.md)
9. [GUI accounts](phase-9-gui-accounts.md)
10. [GUI shell](phase-10-gui-shell.md)

Each phase is one PR when possible. Phase 4 must not merge without phase 2 seed and the key backfill from phase 5’s data rule (backfill SQL can live in `016` or `017` but must be applied before handlers start returning 403). Prefer shipping 4 and 5 in one PR if a split would lock out existing keys.

## Verification

Project-level, after each phase:

```
gofmt -l on touched Go files (empty)
go test -count=1 ./internal/middleware ./internal/operator ./internal/admin ./internal/coreapp ./cmd/setup
make test   # before merge of a cutover PR
```

Phase 4 and 10 additionally: httptest a viewer principal vs `POST /admin/tenants` (403) and `GET /admin/activity-logs` (200 if `logs:read`). Env key both 200.

No browser driver. GUI nav tests render `base.tmpl` with a principal that lacks `tenants:read` and assert the Tenants link is absent.

## Implementation guidance

- Run **how** on `internal/middleware/admin_auth.go` and `internal/coreapp/app.go` before the JSON cutover.
- Run **interrogate** on contested points: backfill of blank keys to superadmin, in-memory env grant, last-superadmin invariant.
- `/deslop` each diff. **unslop** every doc and PR body.
- **show-me-your-work** on cutover PRs.
- Cursor babysit after opening each PR. Do not treat a green CI as “SOC 2 done.”
- **Laziness Protocol:** do not add a plugin nav registry. Sidebar reads the same permission function as the route. Key expiry already exists (`expires_at`, create form, `FindActiveKeyByHash`, 7-day and 1-day emails). Phase 5 keeps it and puts the same field on edit. Duration presets (forever / 30 / 90 / 365 / custom) are optional UX that still write that one timestamp. Do not rebuild expiry.
- **Subtract Before You Add:** stop treating `api_keys.scopes` as admin authorization. Leave the column until a later cleanup if display still needs it, but do not parse it for `/admin/*`.
- **Never Block on the Human** for reversible internals. Do not change the four seeded role names without a product call.
- **Sequence Work into Verifiable Units:** catalog tests before middleware tests before route tests.
- **Prove It Works:** a 403 on the real Gin route, not only `HasPermission` on a struct.

## Organizational track (not code)

Named DB keys for daily use. Env key is break-glass and reviewed when the access log shows `kind=env_key`. Joiner = viewer. Leaver = revoke key / disable account. Temporary access sets `expires_at`; standing access leaves it null. Calendar access review of the roster export. Retention for operator logs longer than end-user informational events. Type I after phases 4–8 are in production. Type II after that period of consistent use.

## Permission catalog

Resources. Actions are `read` and `write` unless noted.

| Resource | Meaning |
|----------|---------|
| `dashboard` | GUI home and stats (`read` only) |
| `tenants` | Tenants JSON and GUI |
| `applications` | Apps |
| `oauth` | OAuth provider configs |
| `oidc` | OIDC clients (JSON `/admin/oidc/...` and GUI) |
| `session_groups` | Cross-app SSO groups |
| `users` | App users, import/export, trusted devices, social/passkey admin actions |
| `sessions` | App-user sessions, revoke |
| `ip_rules` | IP allow/block |
| `end_user_rbac` | App-user roles, permissions, user-role assignment |
| `email` | Servers, templates, types, send/test |
| `logs` | End-user activity log viewer (`read` only) |
| `api_keys` | Create/revoke admin and app keys |
| `webhooks` | Admin webhook CRUD |
| `monitoring` | Health, `GET /metrics` (`read` only) |
| `settings` | System settings |
| `admin_iam` | Operator roles, operator accounts, assignments |

HTTP mapping: GET and safe list/detail/export/preview/check = `read`. POST, PUT, PATCH, DELETE, toggle, import, send, rotate, revoke, unlock = `write`. Form-cancel and confirm-page GETs follow the mutating resource’s `read` (page) vs `write` (the POST/DELETE that follows). Prefer `read` on confirm GET so a viewer who cannot delete also does not need the confirm modal.

## Seeded grants

| Role | Grant |
|------|-------|
| `superadmin` | All resources, all actions, including `admin_iam` |
| `admin` | All except `admin_iam` |
| `support` | `dashboard:read`, `users:read`, `users:write`, `sessions:read`, `sessions:write`, `logs:read` |
| `viewer` | `dashboard:read`, `users:read`, `logs:read`, `monitoring:read` |

System roles cannot be deleted. Seeded permission sets stay frozen on upgrade (`ON CONFLICT DO NOTHING`). Custom roles are extra rows with `is_system=false`.
