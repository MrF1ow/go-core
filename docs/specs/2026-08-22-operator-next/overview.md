# Operator IAM next

**Date:** 2026-08-22
**Status:** plan, not implemented
**Depends on:** leftover IAM on main (PRs 14 through 19)
**Parent leftover:** [2026-08-22-operator-iam-leftover](../2026-08-22-operator-iam-leftover/overview.md) (shipped)

The doors are locked. JSON can list who holds which role, record who changed it, and store allow/deny on operator writes. This directory is the leftover exclusions plus the log work a SOC 2 claim actually needs.

Do not implement from the overview. Implement from the phase files, one impl PR per [Delivery](#delivery) row. Same leftover rule: `internal/sqlcgen` is one writer. `internal/schema.sql` is one writer per wave. `internal/coreapp/app.go` is one writer per wave. Do not stand up Orchestrate. Five impl PRs fit one agent.

[PR #7](https://github.com/MrF1ow/go-core/pull/7) is stale. Close it. Do not implement `docs/specs/2026-08-21-operator-rbac/`.

## Should this be a loop?

Yes for impl PRs after this plan is reviewed. No for a second plan PR per slice. This directory is the spec.

## Context

Operator catalog, seeded roles, `Principal.Has`, JSON `requireOp`, GUI `requireGUI`, sidebar omit, API-key IAM stamp, JSON roster and CSV, `018` evidence tables, access-log writes, IAM events, JSON key role PUT, JSON account create/role/disable, last-superadmin refuse, and `disabled_at` are on main.

`principalForDBKey` still mints `SuperadminPrincipal` when `api_keys.operator_role_id` is null. CSRF still JSON-aborts. There is no `admin_iam` nav row. Operator evidence tables have no retention and no CSV. `ParseAssignedAdminRole` rejects any id that is not one of the four frozen roles.

App-user `roles` / `user_roles` and `activity_logs` stay a different domain. Do not reopen them.

## Scope

Included, across the impl PRs in [Delivery](#delivery):

- Migration `019`: backfill leftover null admin-key roles to viewer, then `CHECK (key_type <> 'admin' OR operator_role_id IS NOT NULL)`
- Delete the fail-open branch. Null admin-key role is 401, same as unknown
- CSV export for access logs and IAM events, same cap as roster
- Age-based cleanup of both evidence tables (365 days)
- CSRF HTML 403 that does not send `X-GUI-Forbidden`
- GUI roster, IAM events, and access logs under one System nav row
- GUI mutations for account create, role change, and disable
- Custom operator roles: stamp, JSON, GUI. No `admin_iam` grants on non-system roles

Excluded, still later:

- Forced must-expire on admin keys
- Per-app operators
- Dashboard stats vs custom roles (stats stay `dashboard:read`)
- IP or UA on `operator_access_logs`
- SOC 2 Type I/II organizational evidence
- Implementing PR #7

## Constraints

- Same catalog. Same `Principal.Has`. No Redis grant store. No `Grants()` lattice.
- Next migration is `019_`. Update `internal/schema.sql` in the same PR. `sqlc.yaml` reads `internal/schema.sql`.
- Fail-open does not run `sqlc generate`. The CHECK needs no new query. Custom roles is the exclusive `sqlc generate` writer (new insert/update/delete queries on tables that already exist).
- JSON operator routes stay on `admin.Handler` in `operator_handler.go`. GUI pages stay on `GUIHandler`. `internal/operator` does not import `admin` or `middleware`.
- Register JSON as `adminRoutes.GET("/operator/...")`. Do not `Group("/operator")` unless the inventory lists that ident.
- GUI deny stays `AbortGUIForbidden` plus `X-GUI-Forbidden: 1`. CSRF must not send that header. `TestCSRFForbiddenDoesNotSendGUIHeader` stays, and it must stop requiring JSON.
- `activity_logs` stays end-user. Operator evidence stays the `018` tables.
- Access insert policy is unchanged: deny always, write allow always, env-key allow always, ordinary read allows skip.
- Last-superadmin still counts enabled GUI accounts with role superadmin.
- No control-ui. Verify with `go test` and httptest.
- App keys keep null `operator_role_id`. Do not `SET NOT NULL` on the whole column.
- Env key stays synthetic superadmin. That is break-glass, not fail-open.

## Alternatives

**A. One plan directory, then a wave of impl PRs (chosen).** The leftover exclusions are no longer unknown. Five spec PRs stall the lock. Sequence work into verifiable units. Separate before serializing `sqlcgen`, `schema.sql`, and `app.go`.

**B. Spec plus impl pair per leftover exclusion.** The user asked to spec the sequence so it can be built. Five plan reviews for one program.

**C. One impl PR for everything.** A reviewer cannot reject CSRF HTML without also rejecting custom roles.

Fail-open, contested leftover:

- **401 on null admin role (chosen).** Same as expired and unknown. The row is not a valid operator.
- 500. Looks like a server bug. It is a bad row.
- Coerce to viewer at auth time. Hides corruption. Roster would lie.

CSRF, contested leftover:

- **HTML body, no `X-GUI-Forbidden`, HTMX stays unswapped (chosen).** GUI shell phase 3 locked that header to IAM deny.
- Reuse `forbidden.tmpl` and send the header. CSRF is not missing a grant.
- Global 4xx swap. GUI shell rejected it.

Retention:

- **`DELETE WHERE at < now() - interval '365 days'` (chosen).** One window. No `expires_at`. Activity logs need per-row expiry because severity splits 365/180/90. Operator tables do not.
- Copy `expires_at` onto both tables. Extra migration and sqlc for no extra policy.
- Export only. Unbounded tables.

Custom `admin_iam` grants:

- **Refuse on non-system roles (chosen).** That is how seeded `admin` is defined.
- Allow. A custom role becomes a second superadmin.
- Cannot-grant-above-self lattice. Leftover rejected `Grants()`.

## Applicable skills

- `go-core` hub, then `admin-gui.md`, `route-map.md`, `data-model.md`, `security.md`
- `how` over `principalForDBKey`, `CSRFMiddleware`, `AbortGUIForbidden`, `ParseAssignedAdminRole`, `web/nav.go`, and `internal/coreapp/app.go` `guiAuth` before those files change
- `interrogate` before marking Fail-open or Custom-roles ready
- `unslop` on docs and PR copy
- `/deslop` before each commit

## Delivery

| PR | Phases | Base | Parallel |
|----|--------|------|----------|
| Plan (this directory) | docs | `main` | no |
| Fail-open | [1](phase-1-fail-open-schema.md), [2](phase-2-fail-open-auth.md) | `main` | with CSRF, with Evidence export |
| Evidence export | [3](phase-3-evidence-export.md) | `main` | with Fail-open, CSRF |
| Evidence retention | [4](phase-4-evidence-retention.md) | Evidence export | serialize with Fail-open if both touch `schema.sql`. They should not |
| CSRF | [5](phase-5-csrf-html.md) | `main` | with Fail-open and Evidence |
| GUI roster | [6](phase-6-gui-nav-roster.md) | `main` | with CSRF if read-only |
| GUI evidence pages | [7](phase-7-gui-evidence-pages.md) | GUI roster | no. Shared `gui_handler` and nav |
| GUI mutations | [8](phase-8-gui-mutations.md) | GUI evidence pages and CSRF | no |
| Custom stamp | [9](phase-9-custom-stamp.md) | Fail-open | exclusive `sqlc generate` |
| Custom roles UI | [10](phase-10-custom-roles-ui.md) | Custom stamp and GUI roster | no |

Close [PR #7](https://github.com/MrF1ow/go-core/pull/7) as soon as this plan is up. No phase file.

## Phases

1. [Fail-open schema](phase-1-fail-open-schema.md)
2. [Fail-open auth](phase-2-fail-open-auth.md)
3. [Evidence export](phase-3-evidence-export.md)
4. [Evidence retention](phase-4-evidence-retention.md)
5. [CSRF HTML](phase-5-csrf-html.md)
6. [GUI nav and roster](phase-6-gui-nav-roster.md)
7. [GUI evidence pages](phase-7-gui-evidence-pages.md)
8. [GUI mutations](phase-8-gui-mutations.md)
9. [Custom role stamp](phase-9-custom-stamp.md)
10. [Custom roles UI](phase-10-custom-roles-ui.md)

Companion: [testing.md](testing.md).

## Verification

```
gofmt -l on touched Go files (empty)
go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/coreapp ./web ./cmd/setup
```

Full merge bar is in [testing.md](testing.md). Each phase has its own check. `make ci` before an impl PR leaves draft.

No control-ui in this environment. Do not claim browser verification.

## Implementation guidance

- Run **how** on `internal/middleware/admin_auth.go` `principalForDBKey`, `internal/middleware/csrf.go`, `internal/middleware/gui_permission.go` `AbortGUIForbidden` / `guiLayoutData`, `internal/operator/iam.go` `ParseAssignedAdminRole`, `web/nav.go`, and `internal/coreapp/app.go` `guiAuth` before the wave that edits them.
- **Foundational thinking.** CHECK and backfill before deleting the fail-open branch. Stamp before custom UI. Nav types before pages.
- **Boundary discipline.** `ParseAssignedAdminRole` stays pure. CSRF middleware writes HTML and aborts. It does not call `Has`. Last-superadmin stays `WouldLeaveLastSuperadmin`.
- **Type system discipline.** Admin keys with null role are unrepresentable after `019`. App keys keep the pointer. `is_system` is the custom vs frozen split. Do not add `is_custom`.
- **Model the domain.** One System nav row for roster, events, and access logs. CSRF is a session-token miss, not an IAM deny. Custom roles with zero grants are deny, not superadmin.
- **Encode lessons in structure.** Pin `019` in `catalog_sql_test.go`. CSRF test asserts HTML and no `X-GUI-Forbidden`. Superadmin nav test asserts the IAM row. `requireGUI` inventory covers new `guiAuth` lines.
- **Build the lever.** httptest through `requireOp` and `requireGUI` on the real Gin group. A unit `Has` is not the merge bar.
- **Laziness protocol.** Reuse `BuildRoster`, `loadRoster`, roster CSV, `guiLayoutData`, `ListOperatorRoles`, `RoleGrants`, and the activity-log batched DELETE shape. Put GUI IAM in `gui_iam.go`. Copy `role_permissions.tmpl` for the grant editor. Do not add `expires_at` on operator tables. Do not add roster SQL. Do not extend `CleanupService`.
- **Experience first.** Nav and roster page land in the same commit. CSRF HTML lands before GUI write CTAs. Empty Email heading rule still holds: no empty IAM heading.
- **Subtract before you add.** Delete the fail-open branch. Do not leave a config flag that restores it.
- **Migrate callers then delete.** `SuperadminPrincipal` for DB keys goes away in phase 2. Env key still uses it.
- **Outcome-oriented execution.** Empty evidence tables after leftover schema-only was fine. Unbounded tables after this plan are not. Dual fail-open/fail-closed is not a shippable middle.
- **Separate before serializing.** Custom stamp owns `sqlc generate`. Fail-open owns `019` and `schema.sql`. GUI roster owns the first `guiAuth` `/operator` lines. `OperatorAccountRole` still uses `IsSystemRoleID` directly. Custom stamp must change that handler too.
- **Never block on the human** for reversible internals. Do not change seeded role names.
- **Interrogate** before marking Fail-open or Custom-roles ready.
- Cursor babysit after each impl PR leaves draft.

## Consumer and maintainer

Consumer. After leftover, a superadmin key can already export the roster over JSON. After Fail-open, a null-role admin key is 401. After Evidence, those JSON lists have CSV and a 365-day cleanup. After GUI roster, a superadmin cookie can review operators in `/gui` without a JSON client. After Custom roles, that cookie can mint a job that is not one of the four frozen names, and it still cannot grant `admin_iam`.

Maintainer. Next impl after this plan is Fail-open, not custom roles. CSRF may be drafted in parallel. Do not start custom roles because they feel like real RBAC. The four frozen roles plus fail-closed plus retained logs are the SOC 2 path.
