# Operator GUI principal

**Date:** 2026-08-21
**Status:** plan, not implemented
**Depends on:** [PR #8](https://github.com/MrF1ow/go-core/pull/8) (merged). Catalog, JSON `RequireOperatorPermission`, key roles, optional expiry.
**Parent plan:** [PR #7](https://github.com/MrF1ow/go-core/pull/7) phases 1–5 shipped. This directory is the next implementation slice, old phase 9, split small.

## Context

Admin JSON is no longer all-powerful. A viewer key gets 403 on `POST /admin/tenants`. The HTMX GUI is still all-powerful. `GUIAuthMiddleware` sets `admin_id` and never builds `operator.Principal`. `KindGUIAccount` and `Principal.AccountID` exist and are unused. `admin_accounts` has no `operator_role_id`. Any logged-in cookie can mint a superadmin JSON key from `/gui/api-keys`.

The original stack put roster, IAM history, and access log before GUI accounts so “who had what” existed before humans got roles. That rationale assumed the GUI was not yet a product door. It already is. A key roster does not close the cookie door. Attaching a principal is the scaffold every later GUI guard, access log, and last-superadmin rule needs.

## Scope

Included:

- JSON leftover levers from the architect review of #8: catalog vs seed check, `/admin`+`/metrics` permission inventory.
- `admin_accounts.operator_role_id` FK, same as keys.
- Existing GUI rows backfill to `superadmin` (they are gods today). New rows default to `viewer`.
- `cmd/setup`: first account `superadmin`, further accounts `viewer`. No `--role` flag.
- `GUIAuthMiddleware` loads `KindGUIAccount` from the account’s role. Redis session stays the admin UUID. Grants load live, same as keys.
- Fail-closed when the session’s account is gone (`ValidateSession` currently can panic on `nil, nil`).
- Null GUI role is 500 after backfill, not silent superadmin.

Excluded (later slices, still the parent plan):

- `RequireOperatorPermission` on `/gui/*` (old phase 10). Principal with no GUI `requireOp` does not change what a cookie can do. That is the following PR, not this one.
- Sidebar hiding.
- HTML/HTMX 403 page. Not needed until GUI `requireOp` exists.
- Roster, IAM history, access log (old phases 6–8). Land after people have roles so the roster is not an empty account list.
- JSON `POST /admin/operator/accounts`, role PUT, last-superadmin invariant, custom roles UI.
- `disabled_at` / leaver flag.
- Per-app operators, env-key replacement, `HasScope` deletion.

## Constraints

- Next migration is `017_`. Update `internal/schema.sql` and run `sqlc generate`. No `IF NOT EXISTS` on new columns. Seed/backfill with explicit UUIDs from `internal/operator/seed_ids.go`.
- Do not import `internal/rbac`.
- Do not put grants in the Redis session blob.
- Do not put `requireOp` on `guiAuth.Use`. Logout and My Account stay session-only.
- `RequireOperatorPermission` stays JSON. Do not reuse its `AbortWithStatusJSON` on HTMX in this slice.
- Cookie path, CSRF, and login ceremony stay as they are.
- No control-ui in this environment. Verify with `go test` and httptest.

## Alternatives

**A. Original order: roster, then history, then access log, then GUI accounts.** Roster of keys is real work and does not change who can mint superadmin. Rejected for this slice. **Outcome-oriented execution.**

**B. Jump straight to GUI `requireOp`.** Middleware would 500 because no principal is attached. Rejected. **Foundational thinking** (principal before guards).

**C. Column + setup + GUI principal, no GUI guards yet (chosen).** Same shape as JSON cutover: identity first, enforcement next PR. Cookie behavior unchanged until phase 10. Tests prove `KindGUIAccount` is on the request.

## Applicable skills

- `go-core` hub, then `admin-gui.md`, `route-map.md`, `data-model.md`, `security.md`
- `how` over `GUIAuthMiddleware` and `cmd/setup` before editing
- `interrogate` before shipping the GUI `requireOp` PR (not this slice)
- `unslop` on docs and PR copy
- `/deslop` before each commit

## Phases

1. [JSON leftover levers](phase-1-json-levers.md)
2. [Account role column](phase-2-account-role-column.md)
3. [Setup CLI roles](phase-3-setup-cli.md)
4. [GUI principal](phase-4-gui-principal.md)

Ship 2–4 in one PR if a split would leave accounts without a role while middleware expects one. Phase 1 can merge alone.

After this directory: GUI shell (`requireOp` on `/gui/*` plus sidebar `Can`), then roster / IAM history / access log from the parent plan.

## Verification

```
gofmt -l on touched Go files (empty)
go test -count=1 ./internal/operator ./internal/middleware ./internal/admin ./cmd/setup
```

Phase 4: httptest a GUI session whose account is `viewer` sets `OperatorPrincipalKey` with `KindGUIAccount` and role `viewer`. Deleted account is a login redirect, not a panic.

## Implementation guidance

- Run **how** on `internal/middleware/gui_auth.go` and `cmd/setup/main.go` before the middleware change.
- **Type system discipline:** after backfill, `operator_role_id` is NOT NULL. Do not copy the API-key null-means-superadmin branch onto accounts.
- **Encode lessons in structure:** the catalog-vs-seed test and the Gin route inventory are the levers from the #8 review. Do not re-state “keep SQL in sync” in a comment and move on.
- **Laziness protocol:** reuse `GrantLookup.RoleGrants`. Do not add a session-cached grant blob.
- **Boundary discipline:** build the principal in `GUIAuthMiddleware`. Handlers keep reading `admin_id` until the shell PR.
- **Sequence work into verifiable units:** schema tests, then setup tests, then middleware tests.
- **Prove it works:** a request through `GUIAuthMiddleware`, not only `NewPrincipal` in a unit test.
- **Never block on the human** for reversible internals. Do not change seeded role names.
- Cursor babysit after opening each PR.
