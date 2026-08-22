# Operator IAM leftover

**Date:** 2026-08-22
**Status:** plan, not implemented
**Depends on:** [PR #12](https://github.com/MrF1ow/go-core/pull/12) (merged). JSON `requireOp` and GUI `requireGUI` are on main.
**Parent:** [PR #7](https://github.com/MrF1ow/go-core/pull/7) old phases 6 through 9. Do not implement those files. This directory replaces them.

The doors are locked. This directory is who has which operator role, who changed it, and which admin calls were allowed or denied.

## Should this be a loop?

Yes for implementation PRs. No for plan PRs. No for an autonomous `/loop until leftover ships` run.

The shell needed a fresh spec ([PR #11](https://github.com/MrF1ow/go-core/pull/11)) because the parent plan was stale. That is done once. Do not write a spec PR per leftover slice.

Do not open one impl PR that mixes roster, schema, access log, and account CRUD. `internal/sqlcgen` is one writer. `internal/coreapp/app.go` is one writer per wave. Four features in one diff hides the merge bar.

Do not stand up Orchestrate for four PRs. That playbook is for a program that outlives one agent.

The loop is a merge-gated wave of Feature playbooks against this spec:

1. This plan PR. Docs only. Stop until it is reviewed.
2. Wave 1, two PRs off `main` in parallel. Roster (no sqlc). Schema `018` (exclusive `sqlc generate`).
3. Wave 2, stacked on schema. Access log, then history plus accounts. They both add `/admin/operator` routes. Stack them. Do not dual-write `app.go`.

After each impl PR merges, start the next Feature from this directory. Do not open another plan PR.

## Context

Operator catalog, seeded roles, `Principal.Has`, JSON enforcement, key expiry, GUI principal attach, and GUI route guards are on main. Extra `cmd/setup` accounts are already viewer. First setup account is superadmin.

There is still no list of who holds which role. Extra GUI operators are still CLI-only. Role assignment is not an event. Permission decisions are not stored. `admin_accounts` has no `disabled_at`. Last superadmin can be demoted in SQL because nothing refuses it.

App-user `roles` / `user_roles` and Session Groups are a different domain. They are already gated. This leftover does not touch them.

## Scope

Included, across the impl PRs named in [Delivery](#delivery):

- JSON roster of admin keys and GUI accounts, plus a synthetic env-key row
- Roster export, same 10_000 row cap as activity-log export
- Migration `018`: `operator_iam_events`, `operator_access_logs`, `admin_accounts.disabled_at`
- Access log writes from `RequireOperatorPermission` and `RequireGUIPermission`
- JSON access-log list
- IAM events on key create, key role change, account create, account role change, account disable, setup CLI create
- JSON `PUT /admin/operator/keys/:id/role`
- JSON operator-account create, role change, disable
- Last-superadmin refuse on demote and disable
- GUI login reject when `disabled_at` is set

Excluded, still later:

- GUI roster pages and an `admin_iam` sidebar row
- Custom operator roles UI
- Flipping null `api_keys.operator_role_id` off fail-open superadmin
- CSRF JSON-to-HTML
- Dashboard stats vs custom roles
- Forced must-expire policy
- Per-app operators

## Constraints

- Same catalog. Same `Principal.Has`. No Redis grant store. No `Grants()` lattice.
- Next migration is `018_`. Update `internal/schema.sql` in the same PR as the migration. `sqlc.yaml` reads `internal/schema.sql`, not `migrations/`.
- Only the schema PR runs `sqlc generate`. Roster must reuse `AdminListApiKeys` and `ListAllAdminAccounts`.
- JSON operator routes live in `internal/admin/operator_handler.go`. `internal/operator` stays a domain package. It does not import `admin` or `middleware`. `coreapp` wires callbacks.
- Register as `adminRoutes.GET("/operator/...")` (and POST, PUT). Do not `Group("/operator")` unless `isOperatorRouteRegistration` lists that ident. The inventory only scans `adminRoutes`, `metricsGroup`, and `adminOIDC`.
- JSON 401 for bad auth. JSON 403 for missing grant. JSON 409 last-superadmin via `c.JSON(http.StatusConflict, dto.ErrorResponse{...})`. `internal/admin` does not import `pkg/errors` for these handlers.
- `activity_logs` stays end-user. Operator evidence is the new tables.
- Access insert is best-effort. A failed insert logs and the request still succeeds.
- Denies always log. Writes always log. Env-key allows always log. Skip ordinary read allows in v1.
- Last-superadmin counts enabled GUI accounts with role superadmin. Keys may all be viewer.
- No control-ui. Verify with `go test` and httptest.
- Do not merge schema without `sqlc generate` and `internal/schema.sql` in the same PR.

## Alternatives

**A. One leftover spec, then a wave loop of impl PRs (chosen).** One review of the leftover design. Parallel only where writers do not overlap. **Sequence work into verifiable units.** **Separate before serializing shared state** (`sqlcgen`, then `app.go`).

**B. Spec plus impl pair per leftover, like #11 and #12 four times.** The design is no longer unknown. Four plan PRs stall the lock.

**C. One impl PR for everything.** Schema, middleware, and account CRUD in one diff. A reviewer cannot reject last-superadmin without also rejecting roster.

**D. Autonomous `/loop` or Orchestrate until leftover IAM ships.** Four PRs are inside one agent's budget. Orchestrate ceremony is the wrong size.

Rejected and still rejected:

- Reusing `activity_logs` for operators
- Logging roster GETs as IAM history
- GUI account CRUD before the events table exists
- Custom roles in this leftover
- Fail-open cutover in this leftover

## Applicable skills

- `go-core` hub, then `route-map.md`, `data-model.md`, `security.md`
- `how` over `RequireOperatorPermission`, `persistAPIKey`, `GUIAuthMiddleware`, and `internal/coreapp/app.go` before those files change
- `interrogate` before marking the schema PR or the accounts PR ready
- `unslop` on docs and PR copy
- `/deslop` before each commit

## Delivery

| PR | Phases | Base | Parallel |
|----|--------|------|----------|
| Plan (this directory) | docs | `main` | no |
| Roster | [1](phase-1-roster-types.md), [2](phase-2-roster-http.md) | `main` | with Schema |
| Schema | [3](phase-3-schema.md), [4](phase-4-sqlc.md) | `main` | with Roster |
| Access log | [5](phase-5-access-log-write.md), [6](phase-6-access-log-http.md) | Schema | no. Stack under accounts |
| History and accounts | [7](phase-7-iam-events.md), [8](phase-8-key-role-json.md), [9](phase-9-operator-accounts.md) | Access log | no |

Roster v1 has no `Disabled` field. The accounts PR adds it after `disabled_at` exists. Do not block roster on schema for that column.

Account CRUD must not merge before the events table exists. Creates you cannot record are gone.

## Phases

1. [Roster types](phase-1-roster-types.md)
2. [Roster HTTP](phase-2-roster-http.md)
3. [Schema 018](phase-3-schema.md)
4. [sqlc queries](phase-4-sqlc.md)
5. [Access log write](phase-5-access-log-write.md)
6. [Access log HTTP](phase-6-access-log-http.md)
7. [IAM events](phase-7-iam-events.md)
8. [JSON key role](phase-8-key-role-json.md)
9. [Operator accounts](phase-9-operator-accounts.md)

Companion: [testing.md](testing.md).

## Verification

```
gofmt -l on touched Go files (empty)
go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/coreapp ./cmd/setup
```

Full leftover merge bar is in [testing.md](testing.md). Each phase has its own check. `make ci` before an impl PR leaves draft.

## Implementation guidance

- Run **how** on `internal/middleware/admin_auth.go` `RequireOperatorPermission`, `internal/admin/gui_handler.go` `persistAPIKey` / `persistAPIKeyUpdate`, `internal/middleware/gui_auth.go`, and `internal/coreapp/app.go` `adminRoutes` before the wave that edits them.
- **Foundational thinking.** Roster row type and event row type before HTTP. Schema before writers that would drop events.
- **Boundary discipline.** Parse JSON into domain types in `operator_handler.go`. Last-superadmin is a pure function over a count. Middleware stays a thin hook that calls `Has` then a nil-safe log func set in `coreapp`. `operator` does not import `admin` or `middleware`.
- **Type system discipline.** `Kind` stays `env_key`, `api_key`, `gui_account`. Event `action` is a closed string set. `disabled_at` nil means enabled. Do not add `is_disabled` next to it.
- **Model the domain.** One `RosterEntry` for keys, accounts, and the env-key row. One `AccessRecord` for JSON and GUI. Do not grow parallel DTOs that drift.
- **Encode lessons in structure.** Pin `018` in `catalog_sql_test.go` the same way `016` and `017` are pinned. Do not write "remember sqlc generate" in a comment.
- **Build the lever.** httptest through `requireOp` on the real Gin group. A unit `Has` on a struct is not the merge bar.
- **Laziness protocol.** Reuse `AdminListApiKeys` and `ListAllAdminAccounts`. Do not add roster SQL. Reuse `ParseAssignedAdminRole` for JSON key role PUT. Do not invent a second stamp function.
- **Experience first.** JSON roster before GUI pages. A review can export CSV. Sidebar without a page is worse than no sidebar row.
- **Outcome-oriented execution.** Wave 2 may sit unmerged until both access log and accounts are ready to land in order. Empty event tables in production after schema-only is acceptable. Account CRUD without events is not.
- **Never block on the human** for reversible internals. Do not change seeded role names.
- **Interrogate** before marking Schema or History-and-accounts ready.
- Cursor babysit after each impl PR leaves draft.

## Consumer and maintainer

Consumer. After Roster, a superadmin key can `GET /admin/operator/roster` and see the env key, DB keys, and GUI accounts. After Accounts, that key can add a viewer operator without `cmd/setup`, and cannot demote the last superadmin. After Access log, env-key writes show up as `kind=env_key`.

Maintainer. Next after this leftover is custom roles UI or the null-key fail-open cutover, each its own spec. Do not fold either into wave 2.
