# Operator IAM remaining

**Date:** 2026-08-23
**Status:** plan, not implemented
**Depends on:** [shipped recap](../2026-08-22-operator-iam.md) on `main` through [PR #31](https://github.com/MrF1ow/go-core/pull/31)

The IAM program is on `main`. This directory is the parked leftovers that are still product work, plus one item that is not.

## Should this be a loop?

Yes for impl PRs after this plan is reviewed. No for a second plan PR per slice. This directory is the spec.

Do not stand up Orchestrate. Three impl waves fit one agent. Per-app is the large one. Interrogate it before the first per-app impl PR.

## Context

Admin keys may have null `expires_at`, which is forever. `parseOptionalExpiresAtKeeping` treats an empty edit as forever. Expired non-null keys are already 401.

`operator_access_logs` stores kind, path, decision, and grant. It does not store client IP or user agent. `maybeLogOperatorAccess` already has `gin.Context`. `activity_logs` already uses `ip_address TEXT` and `user_agent TEXT`.

`admin_accounts` has no `app_id`. `Principal` has no `AppID`. Last-superadmin counts every enabled superadmin GUI account. Admin API keys are platform principals. App-type keys are not operators.

`GET /gui/dashboard/stats` is `requireGUI(dashboard, read)`. Viewer, support, admin, and superadmin all have `dashboard:read`. Custom roles already include or omit that grant through the catalog checkboxes. That is not unfinished work.

## Scope

Included, across the impl PRs in [Delivery](#delivery):

- IP and user agent on `operator_access_logs`, JSON, CSV, and the GUI table
- Forced expiry on admin API keys. Empty create is not forever. Existing null admin keys backfill. App keys and the env key stay as they are
- Per-app GUI operators. Nullable `admin_accounts.app_id`. `Principal.AppID`. Platform routes deny when AppID is set. App-scoped lists filter to that app

Excluded:

- Dashboard stats vs custom roles. Keep `dashboard:read`. Do not invent per-role widgets. A custom role that omits the grant already misses the cards
- SOC 2 Type I/II organizational evidence
- Duration preset buttons beyond a 90-day default on create
- JSON API-key create
- GUI delete of operator accounts
- App-type keys as operator principals
- Per-app **admin API keys**. Admin keys stay platform
- A many-to-many operator-to-app grant matrix
- Changing seeded role names or the frozen catalog

## Constraints

- Same catalog. Same `Principal.Has`. No Redis grant store. No `Grants()` lattice
- Next migrations are `020_` then `021_` then `022_`. Update `internal/schema.sql` in the same PR. `sqlc.yaml` reads `internal/schema.sql`
- Exclusive `sqlc generate` on the PR that adds columns consumed by queries
- `internal/sqlcgen` is one writer. `internal/schema.sql` is one writer per wave. Stack those PRs. Do not dual-write
- JSON operator routes stay on `admin.Handler`. GUI pages stay on `GUIHandler`. `internal/operator` does not import `admin` or `middleware`
- Register JSON as `adminRoutes.GET("/operator/...")`. Do not `Group("/operator")` unless the inventory lists that ident
- Access insert policy is unchanged. Deny always, write allow always, env-key allow always, ordinary read allows skip
- Last-superadmin still counts enabled **platform** GUI superadmins. `app_id IS NULL`
- Superadmin plus a non-null `app_id` is illegal. CHECK it
- No control-ui in this environment. httptest only
- App keys keep nullable `expires_at`. Do not `SET NOT NULL` on the whole column

## Alternatives

IP and UA:

**A. Two TEXT columns, same as `activity_logs` (chosen).** Capture `c.ClientIP()` and `User-Agent` in `maybeLogOperatorAccess`. Truncate UA at 512 bytes at the boundary. Empty string when missing. **Type system discipline.** **Laziness protocol.**

**B. JSON `details` blob.** One column now, every later field is untyped. List and CSV become guesswork.

**C. `inet` plus a separate UA table.** More types, no extra query.

Must-expire:

**A. CHECK plus 365-day backfill, required create, edit cannot clear (chosen).** Create form defaults to now plus 90 days so the required field is filled. Empty create is 400. Empty edit keeps the stored instant. Clearing forever goes away. **Experience first.** **Subtract before you add** (delete the forever path).

**B. Empty create defaults to 90 days with no CHECK.** A raw SQL insert can still mint forever.

**C. Refuse migrate until an operator sets every null.** Not automatable. Blocks deploys.

Per-app:

**A. Nullable `admin_accounts.app_id`, `Principal.AppID`, platform vs app operator (chosen).** One account, one app. Null app is today's platform operator. Admin API keys stay platform. **Redesign from first principles.** **Model the domain.**

**B. Many-to-many operator-app grants.** A second catalog. Last-superadmin and roster explode.

**C. Filter lists only, keep platform routes.** An app operator can still open Settings and Operator IAM. That is not per-app.

Dashboard:

**Keep.** `Has("dashboard","read")` is the whole feature. Custom roles already opt in or out.

## Applicable skills

- `go-core` hub, then `admin-gui.md`, `route-map.md`, `data-model.md`, `security.md`
- `how` over `maybeLogOperatorAccess`, `parseOptionalExpiresAtKeeping`, `principalForDBKey`, `GUIAuthMiddleware`, and `WouldLeaveLastSuperadmin` before those files change
- `interrogate` before marking Must-expire schema or Per-app shape ready
- `unslop` on docs and PR copy
- `/deslop` before each commit

## Delivery

| PR | Phases | Base | Parallel |
|----|--------|------|----------|
| Plan (this directory) | docs | `main` | no |
| Access client | [1](phase-1-access-client-schema.md), [2](phase-2-access-client-http.md) | `main` | no. Exclusive `sqlc generate` |
| Must-expire | [3](phase-3-must-expire-schema.md), [4](phase-4-must-expire-gui.md) | Access client | no. Exclusive `schema.sql` |
| Per-app shape | [5](phase-5-per-app-shape.md) | Must-expire | no. Exclusive `sqlc generate` |
| Per-app session | [6](phase-6-per-app-session.md) | Per-app shape | no |
| Per-app guard | [7](phase-7-per-app-guard.md) | Per-app session | no |
| Per-app lists | [8](phase-8-per-app-lists.md) | Per-app guard | no |

Companion: [per-app-routes.md](per-app-routes.md), [testing.md](testing.md).

Do not start per-app impl until Access client and Must-expire have merged. The first two waves do not need `AppID`.

## Phases

1. [Access-log client schema](phase-1-access-client-schema.md)
2. [Access-log client HTTP](phase-2-access-client-http.md)
3. [Must-expire schema](phase-3-must-expire-schema.md)
4. [Must-expire GUI](phase-4-must-expire-gui.md)
5. [Per-app shape](phase-5-per-app-shape.md)
6. [Per-app session](phase-6-per-app-session.md)
7. [Per-app guard](phase-7-per-app-guard.md)
8. [Per-app lists](phase-8-per-app-lists.md)

## Verification

```
gofmt -l on touched Go files (empty)
go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/coreapp ./web ./cmd/setup
```

Full merge bar is in [testing.md](testing.md). Each phase has its own check. `make ci` before an impl PR leaves draft.

No control-ui. Do not claim browser verification.

## Implementation guidance

- Run **how** on the files named in Applicable skills before the wave that edits them.
- **Foundational thinking.** AccessRecord fields and Principal.AppID before HTTP. CHECK and backfill before deleting the forever path.
- **Boundary discipline.** Truncate and parse at the handler or middleware edge. `AccessRecord` and `Principal` stay trusted.
- **Type system discipline.** Admin keys with null `expires_at` are unrepresentable after `021`. Superadmin with a non-null `app_id` is unrepresentable after `022`. Empty UA is `""`, not nil.
- **Model the domain.** One `AccessRecord` for insert, JSON, CSV, and GUI. Platform vs app is `AppID == nil`, not a second principal type.
- **Encode lessons in structure.** Pin `020`, `021`, `022` in `catalog_sql_test.go`. Inventory still fails a new `/operator` JSON route without `requireOp`.
- **Build the lever.** httptest through `requireOp` and `requireGUI`. A unit `Has` is not the merge bar.
- **Laziness protocol.** Reuse `c.ClientIP()`, `activity_logs` column types, `parseOptionalExpiresAtKeeping` (then replace its forever branch), `WouldLeaveLastSuperadmin` with a platform-only count. Do not add roster SQL.
- **Experience first.** Create form defaults to plus 90 days. Access log table grows two columns, not a details dump.
- **Subtract before you add.** Delete the forever path for admin keys. Do not leave a config flag that restores it.
- **Outcome-oriented execution.** Dual forever/must-expire is not a shippable middle.
- **Separate before serializing.** Access client owns `020` and `sqlc generate`. Must-expire owns `021` and `schema.sql`. Per-app shape owns `022` and `sqlc generate`.
- **Never block on the human** for reversible internals. Do not change seeded role names.
- **Interrogate** before marking Must-expire schema or Per-app shape ready.
- Cursor babysit after each impl PR leaves draft.

## Consumer and maintainer

Consumer. After Access client, a superadmin can see which IP hit `/admin/tenants`. After Must-expire, a new admin key dies on a date. After Per-app, a GUI account bound to one app cannot open Tenants or Operator IAM.

Maintainer. Next impl after this plan is Access client, not per-app. Do not start per-app because it feels like real multi-tenant IAM. IP on the evidence row and keys that cannot live forever are the SOC 2 path.
