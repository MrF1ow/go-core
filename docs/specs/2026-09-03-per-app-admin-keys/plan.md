# Per-app admin keys plan

JSON `/admin` lists filter by `Principal.AppID`. Operator binding uses RESTRICT, not worker CASCADE. Superadmin keys stay platform. Then drop `api_keys_admin_app_id_null`. PR ids in order are json-scope then bind.

## How to read this

One box is one unit of work. Every box names the evidence that checks it. A nested box is a sub-step of the box above it. Check a box only when its evidence exists, a file, a log line, a screenshot, a test run, or a SHA. The body is a how-to. The appendices explain and record.

The program runs `pstack/skills/poteto-mode/playbooks/autopilot-stack.md`. The operator lands both PRs. Owners stop at STACK-READY. Nothing auto-merges.

Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

Stamp AppID and filter JSON before any bound admin key can exist. Do not drop the CHECK in json-scope. Do not mint a bound admin key until bind.

## Program checklist

### Arm the program

- [ ] State the protocol and this plan to the operator, then stop. Start execution only on her explicit go.
- [ ] On her go, arm a `/goal` with this exact text. "docs/specs/2026-09-03-per-app-admin-keys/plan.md. PR ids json-scope then bind. Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. The operator lands the stack. Done when both PRs are verified and appended, JSON /admin lists restrict like GUI, applications cannot delete while an admin key is bound, superadmin keys stay platform, and POST /admin/operator/keys with app_id mints a bound admin key."
- [ ] Read these from trunk at program start. Re-read them at every tick.
  - [ ] `git show origin/main:pstack/skills/poteto-mode/playbooks/autopilot-stack.md`
  - [ ] `git show origin/main:pstack/skills/swarm/SKILL.md`
  - [ ] `git show origin/main:pstack/skills/poteto-mode/playbooks/opening-a-pr.md`
  - [ ] `git show origin/main:docs/specs/2026-09-03-per-app-admin-keys/deferred.md`
  - [ ] `git show origin/main:docs/specs/2026-08-30-operator-key-lifecycle/deferred.md`
  - [ ] `git show origin/main:docs/specs/2026-08-23-operator-iam-remaining/per-app-routes.md`
- [ ] Arm the 30-minute audit tick. In a local session, a real terminal `/loop`. In a cloud root, a cloud-sleeper wake chain. Never leave the cadence to memory.
- [ ] Use this tick prompt, verbatim. "Re-read the execution playbook from trunk and the armed /goal. Audit the operation against both and fix drift in this tick. Probe every active lane and judge progress by side effects only. Stand down a stuck lane and dispatch its replacement now. Then send the operator a status message, whether or not anything changed, with the queue table of PR, owner, state, and head SHA, the verdicts since the last tick, what merged, open operator gates, and blockers."
- [ ] On the operator's hold or stand-down, send every owner a zero-writes order at once.

### Spawn owners

- [ ] Spawn one owner per PR with the full lifecycle the execution playbook names.
- [ ] Follow this dependency graph. Start dependent work only after its parent merges, or base it on the parent branch when the execution playbook stacks.
  - [ ] json-scope first from `main`. Exclusive `sqlc generate` for activity-log queries. No `internal/schema.sql`. Keep `api_keys_admin_app_id_null`.
  - [ ] bind after json-scope. Exclusive `internal/schema.sql`. No `sqlc generate`.
- [ ] Hold the file boundaries. json-scope owns stamp, list restrict, and activity-log SQL. bind owns 024, the CHECK drop, the superadmin CHECK, the restrict trigger, and POST `app_id`.
- [ ] Hold the review gate. bind changes GUI AppDelete copy when an admin key is bound. It waits for the operator's review in chat with screenshots and a video before merge. json-scope does not.

### PR mechanics, for every PR

- [ ] Open the PR ready, never draft, with `gh pr create` and `draft: false`, or with Graphite `gt` for a stack.
- [ ] Run the repo's lint and typecheck once before the PR-facing push. Push with hooks on.
- [ ] Run `/deslop` before each commit and `/no-comments` before review.
- [ ] Triage every Bugbot and security-reviewer comment per `../references/bugbot-triage.md`.
- [ ] Rebase onto current trunk before babysit and again before the merge-ready report.

### Verdict and merge, for every PR

- [ ] At the merge-ready head SHA, run the swarm per `pstack/skills/swarm/SKILL.md`. One gates lane. The ten live lanes from the PR's **Verify, live** block. The perf lane from its **Verify, perf** block. One audit lane that reads the diff and the receipts and distrusts the PR body.
- [ ] Clean only when every lane is `PASS`. Findings go back to the owner. A new head gets a fresh swarm and a fresh verdict.
- [ ] The root appends the PR to the Graphite stack. The operator lands it. Patch-id after restack must match the verdict SHA. Drift goes back through swarm.

### Boot recipe, for every live lane

Each live lane runs on its own cloud VM at the PR head. Drive through httptest. This repo has no control-ui. Save the response body or a page capture at the screenshot path.

- [ ] `git fetch origin <head-branch> && git checkout <head SHA>`.
- [ ] `make docker-dev` if Postgres is down. `make migrate-up`. Wait until `/health` or the test binary starts.
- [ ] Deliver input only through httptest against `internal/coreapp` engines already used in `operator_iam_test.go` and `gui_shell_test.go`. Read-only diagnostics are `curl` of `/health` and the test log.
- [ ] Save every screenshot to `/tmp/swarm-<pr-id>/worker-<n>/<slug>.png` and return the paths with the report.

## Restrict JSON lists and stamp AppID (json-scope)

**Depends on.** None.

**Files.**

- [ ] Edit `internal/operator/scope.go`. Create it if missing.
- [ ] Edit `internal/admin/gui_scope.go`.
- [ ] Edit `internal/middleware/admin_auth.go`.
- [ ] Edit `internal/admin/handler.go`.
- [ ] Edit `internal/log/handler.go`.
- [ ] Edit `internal/log/query_service.go`.
- [ ] Edit `internal/log/repository.go`.
- [ ] Edit `internal/queries/activity_log.sql`.
- [ ] Edit `internal/webhook/handler.go`.
- [ ] Edit `internal/rbac/handler.go`.
- [ ] Edit `internal/admin/gui_scope_test.go`.
- [ ] Edit `internal/coreapp/operator_iam_test.go`.
- [ ] Run `sqlc generate` in this PR only.

**Build.**

- [ ] Put `RestrictAppQuery(bound *uuid.UUID, requested string) string` and `ForeignApp(bound *uuid.UUID, resource uuid.UUID) bool` on `internal/operator`. Same rules as `gui_scope.go`. Bound overwrites the requested app. Foreign is true when bound is set and the resource app differs. Invalid parse is foreign when bound is set.
- [ ] `gui_scope.go` calls those functions. Do not keep a second copy of the rules.
- [ ] `principalForDBKey` copies `foundKey.AppID` the way `GUIAuthMiddleware` copies `account.AppID`. Copy the UUID, do not alias the pointer.
- [ ] JSON 404, not 403, for a foreign app. Same as GUI. Message is generic not-found.
- [ ] App-scoped JSON only. Frozen map is `per-app-routes.md`. Bound already 403s platform resources through `Allows`. Do not filter tenants, applications, admin_iam, email, oidc, settings, monitoring, or session_groups.
- [ ] Users. `ExportUsers` uses `RestrictAppQuery` on `req.AppID`. Bound with empty query exports that app only. `ImportUsers` 404s when `foreignAppID`. Trusted-device handlers load the user and 404 when `foreignApp` on `user.AppID`.
- [ ] Logs. Add `sqlc.narg('app_id')::uuid` to `CountAllActivityLogs`, `ListAllActivityLogs`, and `ExportAllActivityLogs`. Bound forces that UUID. Platform with no query stays unfiltered. Do not add a JSON dashboard or JSON sessions list. Those routes do not exist.
- [ ] OAuth. `UpsertOAuthConfig` 404s when `:id` is foreign.
- [ ] IP rules. Every `/admin/apps/:id/ip-rules` handler 404s when `:id` is foreign.
- [ ] Webhooks. Bound `AdminListEndpoints` calls `ListEndpointsByApp(bound)`, not `ListAllEndpoints`. Path `:app_id` 404s when foreign. Toggle, delete, and deliveries-by-endpoint load the row and 404 when `ep.AppID` is foreign.
- [ ] End-user RBAC. `ListRoles` and `ListUserRoles` use `RestrictAppQuery` on the query `app_id`. Bound with empty `app_id` uses the bound UUID instead of 400. Get, update, delete, and permission writes on a role 404 when `role.AppID` is foreign. Assign and revoke 404 when the body's `app_id` is foreign. `ListPermissions` stays global. The catalog has no `app_id`.
- [ ] Bound `POST /admin/operator/keys` is 403. GUI bound operators cannot mint admin keys. JSON matches. `api_keys` is not a platform resource, so `Allows` is not enough. Check `p.AppID != nil` in the handler.
- [ ] Keep `api_keys_admin_app_id_null`. Do not persist a non-null admin `app_id`.

**You see.**

- [ ] httptest bound JSON principal, AppID injected on the key or context, GET `/admin/activity-logs` returns only that app. GET `/admin/webhooks` returns only that app. GET `/admin/users/export?app_id=<other>` is empty or that app, never the other. GET `/admin/apps/<other>/ip-rules` is 404. GET `/admin/tenants` is 403. Platform superadmin lists still span apps. `catalog_sql_test.go` still contains `api_keys_admin_app_id_null`.

**Verify, unit.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] Injected bound JSON principal covers logs, webhooks, user export, IP rules 404, RBAC foreign role 404, OAuth foreign 404, trusted-device foreign 404, tenants 403, POST `/admin/operator/keys` 403. Platform principal still lists both apps. `principalForDBKey` copies a non-null `AppID` in `admin_auth_test.go`. Run `go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/log ./internal/webhook ./internal/rbac ./internal/coreapp`.

**Verify, live.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. Ten lanes on `grok-4.6-fast-xhigh` at the PR head, per the boot recipe.

json-scope cannot persist a bound admin key. Inject `Principal.AppID` after minting a platform key, or set it on the in-memory key the test engine stores. Do not insert `key_type=admin` with a non-null `app_id`.

- [ ] Lane 1. Bound GET `/admin/activity-logs` with logs in app A and app B. Save `logs-bound.png`. Pass when only A appears.
- [ ] Lane 2. Platform GET `/admin/activity-logs`. Save `logs-platform.png`. Pass when A and B appear.
- [ ] Lane 3. Bound GET `/admin/webhooks`. Save `webhooks-bound.png`. Pass when only A's endpoints appear.
- [ ] Lane 4. Bound GET `/admin/webhooks/apps/<B>`. Save `webhooks-foreign-404.png`. Pass when 404.
- [ ] Lane 5. Bound GET `/admin/users/export?app_id=<B>`. Save `users-export-bound.png`. Pass when the file has no B users.
- [ ] Lane 6. Bound GET `/admin/apps/<B>/ip-rules`. Save `ip-rules-404.png`. Pass when 404.
- [ ] Lane 7. Bound GET `/admin/rbac/roles?app_id=<B>`. Save `rbac-404.png`. Pass when 404 or empty for A only, never B's roles.
- [ ] Lane 8. Bound GET `/admin/tenants`. Save `tenants-403.png`. Pass when 403.
- [ ] Lane 9. Bound POST `/admin/operator/keys`. Save `mint-bound-403.png`. Pass when 403 and no row.
- [ ] Lane 10. Platform GET `/admin/webhooks`. Save `webhooks-platform.png`. Pass when A and B appear.

**Verify, perf.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] Metric. httptest wall time of bound GET `/admin/activity-logs`.
- [ ] Probe. Time that handler ten times at trunk (unfiltered 200) and at the head (filtered 200), interleaved. Also time platform GET `/admin/activity-logs` at both SHAs as a control.
- [ ] Baseline. Record the trunk unfiltered median first.
- [ ] Rule. Head bound median must not exceed twice the trunk median.

**Review gate.** None. json-scope is not review-gated.

**Merge.**

- [ ] Root's clean verdict at the exact head SHA.
- [ ] Bugbot triage done.
- [ ] Rebased onto current trunk after the verdict, patch-id unchanged.
- [ ] The root appends json-scope to the Graphite stack. The operator lands it.

## Bind admin keys and drop the CHECK (bind)

**Depends on.** json-scope.

**Files.**

- [ ] Create `migrations/024_admin_key_app_bind.sql`.
- [ ] Edit `internal/schema.sql`.
- [ ] Edit `internal/operator/catalog_sql_test.go`.
- [ ] Edit `internal/admin/operator_handler.go`.
- [ ] Edit `internal/admin/gui_handler.go` `AppDelete` only.
- [ ] Edit `internal/coreapp/operator_iam_test.go`.
- [ ] Edit `docs/api-endpoints.md`.
- [ ] Run `make swag-init`. Do not hand-edit `docs/swagger.yaml`.

**Build.**

- [ ] Keep `api_keys.app_id REFERENCES applications(id) ON DELETE CASCADE`. That is the worker FK. Do not add a second column.
- [ ] Add `applications_admin_keys_restrict` BEFORE DELETE ON `applications`. If a `key_type=admin` row has that `app_id`, RAISE. Worker keys still CASCADE. Pin the name in `catalog_sql_test.go` next to the `023` pins.
- [ ] Drop `api_keys_admin_app_id_null` from the table and from the `023` schema pin list. Add `api_keys_admin_superadmin_is_platform` `CHECK (key_type <> 'admin' OR operator_role_id <> '<RoleIDSuperadmin>'::uuid OR app_id IS NULL)`. Same UUID `gui_scope` already uses from `operator.RoleIDSuperadmin`.
- [ ] `createOperatorKeyRequest` gains `AppID string json:"app_id"`. Empty is platform. Invalid UUID is 400. Superadmin role plus non-null `app_id` is 400 before insert. `key_type` in the body stays ignored or 400. Persist `app_id` on the admin row. 201 includes `app_id`.
- [ ] Bound POST `/admin/operator/keys` stays 403. Only a platform principal mints a bound admin key.
- [ ] GUI `ApiKeyCreate` still refuses admin keys for bound operators and still stores null `app_id` for platform admin keys. No GUI picker.
- [ ] `AppDelete` maps the restrict raise to a 409 HTML fragment that says the application has bound admin keys. Do not leave a generic 500.
- [ ] Rewrite `TestOperatorCreateKey_IgnoresAppTypeAndAppID`. `key_type=app` still must not insert an app key. Posted `app_id` now persists on an admin row.

**You see.**

- [ ] Platform POST `/admin/operator/keys` with `app_id` returns 201, null `key_type` app, stored `app_id` set, non-superadmin role. That key GET `/admin/activity-logs` sees only that app and GET `/admin/tenants` is 403. POST with superadmin role and `app_id` is 400. `DELETE FROM applications` with a bound admin key raises. `DELETE FROM applications` with only app-type keys succeeds and those keys are gone.

**Verify, unit.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] `TestOperatorCreateKey_` covers 201 with `app_id`, 201 without `app_id` stays null, 400 superadmin plus `app_id`, 403 bound mint, ignore or reject `key_type=app`. Trigger test fails application delete while an admin key is bound and allows delete when only app keys exist. CHECK test fails a superadmin admin key with `app_id`. `catalog_sql_test.go` pins `024`, `applications_admin_keys_restrict`, `api_keys_admin_superadmin_is_platform`, and the absence of `api_keys_admin_app_id_null` in `schema.sql`. Run `go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/coreapp`.

**Verify, live.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. Ten lanes on `grok-4.6-fast-xhigh` at the PR head, per the boot recipe.

- [ ] Lane 1. Platform mint bound viewer key. Save `bind-mint-201.png`. Pass when 201, `ak_` secret, stored `app_id` set.
- [ ] Lane 2. That key GET `/admin/activity-logs`. Save `bind-logs.png`. Pass when only the bound app's logs appear.
- [ ] Lane 3. That key GET `/admin/tenants`. Save `bind-tenants-403.png`. Pass when 403.
- [ ] Lane 4. That key GET `/admin/apps/<other>/ip-rules`. Save `bind-ip-404.png`. Pass when 404.
- [ ] Lane 5. Platform mint with empty `app_id`. Save `bind-platform.png`. Pass when stored `app_id` is null and `/admin/tenants` is 200.
- [ ] Lane 6. Platform mint superadmin plus `app_id`. Save `bind-superadmin-400.png`. Pass when 400 and no row.
- [ ] Lane 7. Bound key POST `/admin/operator/keys`. Save `bind-cannot-mint.png`. Pass when 403.
- [ ] Lane 8. GUI AppDelete while a bound admin key exists. Save `bind-app-delete-409.png`. Pass when 409 or the bound-keys fragment, app row remains.
- [ ] Lane 9. Raw `DELETE FROM applications` with only app-type keys. Save `bind-worker-cascade.png`. Pass when the app and those keys are gone.
- [ ] Lane 10. Replay GET of the bound key id. Save `bind-secret-once.png`. Pass when the raw secret is absent.

**Verify, perf.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] Metric. httptest wall time of platform POST `/admin/operator/keys` with `app_id`.
- [ ] Probe. Time that handler ten times at json-scope HEAD (201, null `app_id`) and at this head (201, set `app_id`), interleaved. Also time POST without `app_id` at both SHAs as a control.
- [ ] Baseline. Record the json-scope-HEAD mint-without-`app_id` median first.
- [ ] Rule. Head mint-with-`app_id` median must not exceed twice that baseline.

**Review gate.** The operator reviews before merge.

- [ ] Copy lane 8 screenshots into `/tmp/swarm-bind/review/bind-review-app-delete.png`.
- [ ] Record a 30 to 60 second video of GUI application delete blocked by a bound admin key. Save it as `/tmp/swarm-bind/review/bind-review.mp4`.
- [ ] Post the screenshots and the video in chat. Stop at merge-ready. Wait for the operator's click.

**Merge.**

- [ ] Root's clean verdict at the exact head SHA.
- [ ] Bugbot triage done.
- [ ] Rebased onto current trunk after the verdict, patch-id unchanged.
- [ ] The root appends bind after json-scope. The operator lands the stack.

## Close the program

- [ ] Every box above is checked with its evidence.
- [ ] Reply to the operator with the report the execution playbook names.
- [ ] Deferred items still live only in `docs/specs/2026-09-03-per-app-admin-keys/deferred.md`.

## Appendix A. Prototype evidence

No prototype ran. Open questions that a run did not settle.

JSON 404 vs empty list on `GET /admin/rbac/roles?app_id=<foreign>`. Plan prefers 404 so the operator cannot probe. Unproven whether existing 400-for-missing-`app_id` clients break when bound omits `app_id` and the handler fills it.

httptest is the live surface. Unproven against a real browser because this environment has no control-ui. bind review still needs a real screen capture when a human is present.

## Appendix B. Alternatives rejected

Two columns, `operator_app_id` RESTRICT plus `app_id` CASCADE. PostgreSQL cannot attach two ON DELETE actions to one column. A second column is a second catalog. Interrogate of trigger vs two columns, 2026-09-03.

Replace CASCADE with RESTRICT for every `api_keys` row, then delete worker keys in `DeleteApp`. A raw SQL app delete would then fail for worker keys too. Worker CASCADE stays on the FK.

Drop the CHECK in json-scope. A bound row plus stamp without list filters is a cross-app read. All three gates in one program, CHECK drop in bind only.

GUI mint of bound admin keys. Bound GUI already cannot create admin keys. JSON is the automation path. Unlock only for a named human who cannot use JSON.

JSON mint of app-type keys on `POST /admin/operator/keys`. Bind owns `app_id` on that body. Empty is platform admin. Set is bound admin. Posted `key_type=app` still inserts `key_type=admin`. Worker JSON mint, if it ever exists, is a new route.

`Allows` as the only bound mint gate. `api_keys` is app-scoped, so a bound key with write would mint a platform admin key. Handler 403 when `p.AppID != nil`.

## Appendix C. Risks

json-scope. Exclusive `sqlc generate` for `activity_log.sql`. bind must not write `internal/sqlcgen`.

bind. Exclusive `schema.sql`. json-scope must not drop `api_keys_admin_app_id_null`.

bind. Trigger syntax `EXECUTE FUNCTION` vs `EXECUTE PROCEDURE`. Pin the syntax 023 already uses.

json-scope. Injected AppID in tests is not a DB bound key. bind live lanes are the first real bound admin keys. Do not call json-scope verified against a persisted bound key.

AppDelete. Restrict raise must not become a 500. Map it in the GUI handler.

POST body `app_id`. Inventory test still fails a new `/operator` JSON route without `requireOp` on the same registration. Do not `Group("/operator")`.

## Appendix D. Links and reading list

Read before json-scope. `how` on `principalForDBKey`, `gui_scope.go`, `GetAllActivityLogs`, `AdminListEndpoints`, `ListRoles`, `ExportUsers`. `docs/specs/2026-08-23-operator-iam-remaining/per-app-routes.md`. `internal/middleware/gui_auth.go` AppID copy.

Read before bind. `how` on `OperatorCreateKey`, `persistOperatorAPIKey`, `AppDelete`, `TestOperatorCreateKey_IgnoresAppTypeAndAppID`. `internal/operator/catalog_sql_test.go` `023` pins. `migrations/022_admin_account_app.sql`.

json-scope does not need interrogate. bind needs `how` first. Interrogate bind only if the trigger vs two-column split is contested.

Trail per `pstack/skills/show-me-your-work/SKILL.md`. Owners keep `decisions.tsv` uncommitted.

Defer list. `docs/specs/2026-09-03-per-app-admin-keys/deferred.md`.
