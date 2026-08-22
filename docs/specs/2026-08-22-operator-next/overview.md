# Operator IAM next

**Date:** 2026-08-22
**Status:** sequence, not an implementation spec
**Depends on:** leftover IAM on main ([#14](https://github.com/MrF1ow/go-core/pull/14)–[#19](https://github.com/MrF1ow/go-core/pull/19))
**Parent leftover:** [2026-08-22-operator-iam-leftover](../2026-08-22-operator-iam-leftover/overview.md) (shipped)

The doors are locked. JSON can list who holds which role, record who changed it, and store allow/deny on operator writes. This directory is the order of the leftover exclusions plus the logging work SOC 2 actually needs.

Do not implement from this file. Each row below is its own spec PR, then its own impl PR. Same rule as leftover: schema exclusive `sqlc generate`, no mega-diff.

## What is already on main

Operator catalog, seeded roles, `Principal.Has`, JSON `requireOp`, GUI `requireGUI`, sidebar omit, API-key IAM stamp, JSON roster and CSV export, `018` evidence tables, access-log writes, IAM events, JSON key role PUT, JSON account create/role/disable, last-superadmin refuse, `disabled_at` on GUI login and roster.

End-user `activity_logs` already have severity, retention, cleanup, IP, UA, anomaly detection, and GUI export. App-user `roles` / `user_roles` are a different domain and already gated. This sequence does not reopen them.

## What this is not

[PR #7](https://github.com/MrF1ow/go-core/pull/7) is a stale ten-phase plan. Leftover replaced its phases 6–9. GUI shell replaced phase 10. Close it. Do not implement `docs/specs/2026-08-21-operator-rbac/`. Do not write a second copy of leftover.

SOC 2 Type I/II organizational evidence (policies, HR, vendor reviews) is out. This sequence is the product controls an auditor will ask the module to show: unique privileged identities, least privilege that is actually fail-closed, access reviews, privileged-action history, and retained security logs.

Still later, not in this sequence:

- Forced must-expire on admin keys (optional `expires_at` stays; forever remains the default)
- Per-app operators
- Dashboard stats vs custom roles (stats stay `dashboard:read` until a dedicated spec after custom roles)
- IP/UA columns on operator access logs (v1 `AccessRecord` has none; do not sneak them into retention)

## Order

Serial for humans. Note where writers do not overlap if two agents run at once.

| # | Spec | Why this slot | SOC 2 / RBAC |
|---|------|---------------|--------------|
| 0 | Close PR #7 | Housekeeping. No spec. Comment pointing at #8–#19 and this directory. | Stops a stale plan from shipping a second catalog. |
| 1 | Null-key fail-open cutover | Last hole that makes `Has` a lie. JSON roster already shows empty role names on null `api_keys.operator_role_id`. `principalForDBKey` still mints `SuperadminPrincipal`. | CC6 least privilege. A null admin key is standing superadmin. |
| 2 | Operator evidence retention and export | Access logs and IAM events are append-only with no retention, no cleanup, no CSV. Auditors cannot keep or produce them the way they can `activity_logs`. No new GUI. | CC7 log retention and evidence export. |
| 3 | CSRF JSON-to-HTML | Independent of IAM. HTMX 2.0.4 does not swap 4xx. CSRF still `AbortWithStatusJSON`. GUI IAM mutations will POST through this middleware. | Existing CSRF control. HTML so a failed GUI write is a page, not a JSON blob. |
| 4 | GUI IAM: roster, `admin_iam` nav, events, access logs | Leftover said JSON before GUI, and no sidebar without a page. JSON is on main. One spec for the whole `admin_iam` section, not a nav row plus three later page specs. | Access reviews and log review in the GUI. |
| 5 | Custom operator roles UI | Catalog already has `is_system`. Custom roles are useless while null keys are superadmin, and they live under the `admin_iam` nav from spec 4. | Least privilege beyond the four frozen roles. Not a Type I blocker if those four are documented. |

Do not start 5 before 1. Do not start 4 mutations before 3. Spec 4 may ship read-only pages before 3 if mutations wait for CSRF HTML.

Spec 1 and spec 2 can overlap if 1 does not run `sqlc generate` and 2's first PR is export-only. Retention that adds columns is the exclusive sqlc writer. Do not dual-write `internal/schema.sql`.

Spec 3 can overlap 1 and 2. It touches `internal/middleware/csrf.go` and templates. It must not send `X-GUI-Forbidden`. That header is IAM deny, not CSRF. GUI shell phase 3 already locked that.

## Spec 1 — Null-key fail-open cutover

`internal/middleware/admin_auth.go` `principalForDBKey`: null `OperatorRoleID` is still in-memory superadmin. Migration `016` backfilled existing admin keys. New GUI creates stamp viewer. The branch remains for rows that slipped through or for a future writer that forgets the stamp.

The spec must decide, in writing:

- After cutover, null admin-key role is 401, not 500, not superadmin.
- App keys keep null `operator_role_id`. Do not `SET NOT NULL` on the whole column.
- Backfill remaining `key_type = 'admin' AND operator_role_id IS NULL` before any CHECK.
- Roster empty role name goes away for admin keys. Treat a leftover empty name as a test failure.
- Env key stays synthetic superadmin. That is break-glass, not fail-open.

Do not fold custom roles or GUI into this spec.

## Spec 2 — Operator evidence retention and export

`activity_logs` already document retention (365/180/90) and export. `operator_access_logs` and `operator_iam_events` do not.

The spec must decide, in writing:

- Retention days for IAM events vs access logs. Default both to the critical activity-log window (365) unless a reason is written.
- Cleanup reuses the activity-log worker pattern or a sibling. Failed delete logs. No row updates. Append-only stays append-only.
- CSV export on `GET /admin/operator/access-logs/export` and `GET /admin/operator/iam-events/export`. Same 10_000 cap and `X-Export-Truncated` as roster.
- Document the v1 insert policy in [activity-logging.md](../../activity-logging.md) or a sibling: deny always, write allow always, env-key allow always, ordinary read allows skip.
- Do not log roster GETs as IAM history. That leftover reject still holds.
- Do not add IP/UA in this spec.

Empty tables in production after leftover schema-only was acceptable. Unbounded tables without a retention number are not acceptable for a SOC 2 claim.

## Spec 3 — CSRF JSON-to-HTML

GUI deny is already HTML plus `X-GUI-Forbidden: 1`. CSRF and settings env-lock must stay off that header. The JSON body is the leftover.

The spec must decide, in writing:

- HTML body for missing and invalid CSRF. Same username / CSRF / `Can` / `NavGroups` ingredients `page(c)` uses, or a dedicated small template. Do not name it `"error"`.
- Typed URL POST gets a real page. HTMX POST without `X-GUI-Forbidden` stays unswapped unless the spec picks a different, tested path (toast, retarget). Do not turn on global 4xx swap.
- Settings env-lock stays its own 403. Do not reuse the CSRF template for env-lock unless the spec says they are the same user-visible class.
- JSON `/admin` is unchanged.

Do not start GUI IAM write CTAs until this ships, or keep those CTAs out of spec 4's first impl PR.

## Spec 4 — GUI IAM

`web/nav.go` still asserts `admin_iam` is absent. JSON already has roster, access logs, IAM events, account CRUD, and key role PUT.

One spec. Pages under `/gui/operator/...` (or a single `/gui/iam` with HTMX tabs). `requireGUI(admin_iam, read|write)` on each registration. Inventory scan still has to see them.

The spec must decide, in writing:

- Nav: one System row, label like "Operator IAM", resource `admin_iam`, action `read`. Superadmin sees it. Admin does not. Viewer does not. Update `nav_test.go`.
- Roster page reuses `BuildRoster` / `loadRoster`. Do not add roster SQL.
- Events page and access-log page reuse the JSON list types. Filters that JSON already has (decision, target key/account).
- Read-only is a legal first impl PR. Mutations (create viewer account, change role, disable) are a second impl PR in the same spec, after CSRF HTML.
- Write CTA omit via `Can`. Tampered POST still 403 from `requireGUI`.
- Last-superadmin 409 is HTML, not JSON, on GUI mutations.
- Do not add custom-role create here.

Experience first: a sidebar row that 404s is worse than no row. Land page plus nav in the same commit.

## Spec 5 — Custom operator roles UI

Schema already has `operator_roles.is_system` and `operator_role_permissions`. Frozen IDs stay. `ParseAssignedAdminRole` currently stamps system roles. Custom roles need a stamp that is not "any UUID the poster likes."

The spec must decide, in writing:

- Create/edit/delete only non-system roles. System names are not writable.
- Grant editor is the frozen catalog checkboxes, not free-text `resource:action`.
- `admin_iam:*` on a custom role is allowed only if the spec says so. Default no: that is how `admin` is already defined.
- Assign custom roles through the same GUI/JSON paths as system roles once the stamp accepts them.
- Dashboard stats stay `dashboard:read`. Do not invent per-role widgets here.
- Fail-open is already gone. A custom role with zero grants is deny, not superadmin.

## Delivery

| Step | Artifact | Base |
|------|----------|------|
| 0 | Close [PR #7](https://github.com/MrF1ow/go-core/pull/7) | n/a |
| 1a | Fail-open spec PR | `main` |
| 1b | Fail-open impl PR | 1a |
| 2a | Evidence retention/export spec PR | `main` |
| 2b | Evidence impl PR | 2a (stack under 1b if both touch `schema.sql`) |
| 3a | CSRF spec PR | `main` |
| 3b | CSRF impl PR | 3a |
| 4a | GUI IAM spec PR | `main` |
| 4b | GUI IAM read-only impl | 4a, after leftover JSON (already main) |
| 4c | GUI IAM mutations impl | 4b and 3b |
| 5a | Custom roles spec PR | after 4b so the nav exists |
| 5b | Custom roles impl PR | 5a and 1b |

Do not open one spec that covers 1–5. Leftover already proved a reviewer cannot reject fail-open without also rejecting roster when they share a diff.

## Verification

Each later spec owns its merge bar. This sequence is done when 1b has merged (fail-closed keys) and 2b has merged (retained, exportable operator evidence). GUI and custom roles make that operable. They are not the SOC 2 log control.

## Consumer and maintainer

Consumer. After leftover, a superadmin key can already export the roster and list access logs over JSON. After spec 1, a null-role admin key is 401. After spec 2, those JSON lists have CSV and a retention number. After spec 4, a superadmin cookie can do the same review in `/gui` without a JSON client.

Maintainer. Next spec to write is fail-open. Then evidence retention/export. CSRF may be drafted in parallel. Do not start custom roles because they feel like "real RBAC." The four frozen roles plus fail-closed plus retained logs are the SOC 2 path.
