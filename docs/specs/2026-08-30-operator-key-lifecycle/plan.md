# Operator key lifecycle plan

JSON mint of platform admin keys. One-way revoke and disable. No row drop. Consumers automate mint. Maintainers lose the delete path. Rows outlive the credential. PR ids in order are lifecycle then json-mint.

## How to read this

One box is one unit of work. Every box names the evidence that checks it. A nested box is a sub-step of the box above it. Check a box only when its evidence exists, a file, a log line, a screenshot, a test run, or a SHA. The body is a how-to. The appendices explain and record.

The program runs `pstack/skills/poteto-mode/playbooks/autopilot-stack.md`. The operator lands both PRs. Owners stop at STACK-READY. Nothing auto-merges.

Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

## Program checklist

### Arm the program

- [ ] State the protocol and this plan to the operator, then stop. Start execution only on her explicit go.
- [ ] On her go, arm a `/goal` with this exact text. "docs/specs/2026-08-30-operator-key-lifecycle/plan.md. PR ids lifecycle then json-mint. Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. The operator lands the stack. Done when both PRs are verified and appended, key delete is gone, disable cannot clear, and POST /admin/operator/keys mints a platform admin key."
- [ ] Read these from trunk at program start. Re-read them at every tick.
  - [ ] `git show origin/main:pstack/skills/poteto-mode/playbooks/autopilot-stack.md`
  - [ ] `git show origin/main:pstack/skills/swarm/SKILL.md`
  - [ ] `git show origin/main:pstack/skills/poteto-mode/playbooks/opening-a-pr.md`
  - [ ] `git show origin/main:docs/specs/2026-08-30-operator-key-lifecycle/deferred.md`
  - [ ] `git show origin/main:docs/specs/2026-08-23-operator-iam-remaining/overview.md`
- [ ] Arm the 30-minute audit tick. In a local session, a real terminal `/loop`. In a cloud root, a cloud-sleeper wake chain. Never leave the cadence to memory.
- [ ] Use this tick prompt, verbatim. "Re-read the execution playbook from trunk and the armed /goal. Audit the operation against both and fix drift in this tick. Probe every active lane and judge progress by side effects only. Stand down a stuck lane and dispatch its replacement now. Then send the operator a status message, whether or not anything changed, with the queue table of PR, owner, state, and head SHA, the verdicts since the last tick, what merged, open operator gates, and blockers."
- [ ] On the operator's hold or stand-down, send every owner a zero-writes order at once.

### Spawn owners

- [ ] Spawn one owner per PR with the full lifecycle the execution playbook names.
- [ ] Follow this dependency graph. Start dependent work only after its parent merges, or base it on the parent branch when the execution playbook stacks.
  - [ ] lifecycle first from `main`. Exclusive `internal/schema.sql` and `sqlc generate`.
  - [ ] json-mint after lifecycle. No schema.sql. No sqlc generate.
- [ ] Hold the file boundaries. lifecycle owns delete, revoke triggers, and disable lock. json-mint owns `OperatorCreateKey` and the POST registration.
- [ ] Hold the review gate. lifecycle changes the API Keys GUI. It waits for the operator's review in chat with screenshots and a video before merge. json-mint does not.

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

## Remove key delete and lock disable (lifecycle)

**Depends on.** None.

**Files.**

- [ ] Edit `internal/coreapp/app.go`.
- [ ] Edit `internal/admin/gui_handler.go`.
- [ ] Edit `internal/admin/repository.go`.
- [ ] Edit `internal/admin/account_repository.go`.
- [ ] Edit `internal/queries/admin.sql`.
- [ ] Edit `web/templates/partials/api_key_list.tmpl`.
- [ ] Delete `web/templates/partials/api_key_delete_confirm.tmpl`.
- [ ] Create `migrations/023_operator_one_way_revoke.sql`.
- [ ] Edit `internal/schema.sql`.
- [ ] Edit `internal/operator/catalog_sql_test.go`.
- [ ] Edit `internal/coreapp/operator_iam_test.go`.
- [ ] Edit `internal/admin/gui_iam.go` only if disable can still pass a nil time.

**Build.**

- [ ] Drop `DELETE /gui/api-keys/:id` and `GET /gui/api-keys/:id/delete` from `app.go`. Remove `ApiKeyDelete` and `ApiKeyDeleteConfirm`. Remove the trash button. Leave revoke.
- [ ] Delete `AdminDeleteApiKey` from `internal/queries/admin.sql`. Run `sqlc generate` in this PR only.
- [ ] `SetDisabledAt` returns an error when the row already has `disabled_at` and the new value is null.
- [ ] Add triggers in `023`. `admin_accounts` cannot clear `disabled_at`. `api_keys` cannot set `is_revoked` back to false. Pin both names in `catalog_sql_test.go` next to the `022` pins. Keep `api_keys_admin_app_id_null`.

**You see.**

- [ ] API Keys page has Revoke and no Delete. httptest DELETE of the old path is 404. A disabled account still 204s a second disable. A raw SQL `UPDATE admin_accounts SET disabled_at = NULL` fails the trigger.

**Verify, unit.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] `internal/coreapp` httptest DELETE `/gui/api-keys/:id` is 404. `SetDisabledAt` nil after disable errors. Trigger tests in `internal/operator` or `internal/coreapp` fail a clear of `disabled_at` and an un-revoke. Run `go test -count=1 ./internal/admin ./internal/operator ./internal/coreapp ./web`.

**Verify, live.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. Ten lanes on `grok-4.6-fast-xhigh` at the PR head, per the boot recipe.

- [ ] Lane 1. Superadmin GUI lists keys. Save `keys-list.png`. Pass when the trash button is absent and Revoke remains on an active key.
- [ ] Lane 2. Superadmin GET delete confirm. Save `delete-confirm-404.png`. Pass when `/gui/api-keys/:id/delete` is 404.
- [ ] Lane 3. Superadmin DELETE the old path. Save `delete-404.png`. Pass when status is 404 and the key row still exists.
- [ ] Lane 4. Superadmin revoke. Save `revoke-ok.png`. Pass when `is_revoked` is true and the list shows Revoked.
- [ ] Lane 5. Viewer GUI keys page. Save `viewer-keys.png`. Pass when write buttons are hidden and no delete control appears.
- [ ] Lane 6. Bound operator keys page. Save `bound-keys.png`. Pass when app keys list and no admin-key delete appears.
- [ ] Lane 7. Disable a non-last viewer account. Save `disable-ok.png`. Pass when roster shows Disabled and login cookie is bounced.
- [ ] Lane 8. Second disable. Save `disable-idempotent.png`. Pass when 204 or success and one IAM disable event.
- [ ] Lane 9. Last platform superadmin disable. Save `last-superadmin.png`. Pass when 409 and the account stays enabled.
- [ ] Lane 10. Un-revoke SQL. Save `unrevoke-blocked.png`. Pass when `UPDATE api_keys SET is_revoked = false` raises and the row stays revoked.

**Verify, perf.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] Metric. httptest wall time of `GET /gui/api-keys` for a superadmin cookie.
- [ ] Probe. `go test -count=1 -bench=Benchmark` is not required. Time the existing list httptest ten times at trunk and at the head, interleaved.
- [ ] Baseline. Record the trunk median first.
- [ ] Rule. Head median must not exceed twice the trunk median.

**Review gate.** The operator reviews before merge.

- [ ] Copy lane 1 screenshots into `/tmp/swarm-lifecycle/review/lifecycle-review-keys-list.png`.
- [ ] Record a 30 to 60 second video of revoke without a delete button on a lane VM. Save it as `/tmp/swarm-lifecycle/review/lifecycle-review.mp4`.
- [ ] Post the screenshots and the video in chat. Stop at merge-ready. Wait for the operator's click.

**Merge.**

- [ ] Root's clean verdict at the exact head SHA.
- [ ] Bugbot triage done.
- [ ] Rebased onto current trunk after the verdict, patch-id unchanged.
- [ ] The root appends lifecycle to the Graphite stack. The operator lands it.

## Mint platform admin keys over JSON (json-mint)

**Depends on.** lifecycle.

**Files.**

- [ ] Edit `internal/admin/operator_handler.go`.
- [ ] Edit `internal/coreapp/app.go`.
- [ ] Edit `internal/coreapp/operator_iam_test.go`.
- [ ] Edit `docs/api-endpoints.md`.
- [ ] Run `make swag-init`. Do not hand-edit `docs/swagger.yaml`.

**Build.**

- [ ] Add `OperatorCreateKey` on `admin.Handler`. Call `GenerateApiKey("admin")`, `parseOptionalExpiresAt` with required true, `ParseAssignedAdminRole`, `Repo.CreateApiKey` with `app_id` null. Mirror GUI `persistAPIKey` through a `CreateAPIKey` callback already used in tests.
- [ ] Register `adminRoutes.POST("/operator/keys", requireOp(operator.ResAPIKeys, operator.ActionWrite), ...)`. Same statement as `requireOp`. Do not `Group("/operator")`.
- [ ] Body fields are `name`, `description`, `operator_role_id`, `expires_at`. Empty role is viewer. Non-viewer without `admin_iam:write` is 403. Missing expiry is 400. `key_type` in the body is ignored or 400. Never persist a non-null `app_id`.
- [ ] 201 JSON includes the raw secret once, plus id, prefix, suffix, role, expiry. Write `create_principal` on `operator_iam_events`.
- [ ] Run `how` on `GenerateApiKey`, `ParseAssignedAdminRole`, and `persistAPIKey` before the owner writes.

**You see.**

- [ ] `POST /admin/operator/keys` with a superadmin env or DB key returns 201, a raw `ak_` secret, and a row with null `app_id` and non-null `expires_at`. A second GET of that id never returns the raw secret. Viewer without `api_keys:write` is 403. App-type body does not insert an app key.

**Verify, unit.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] `TestOperatorCreateKey_` in `internal/coreapp/operator_iam_test.go` covers 201, secret once, required expiry, viewer default, 403 for support without write, 403 for non-viewer role without `admin_iam:write`, ignore or reject `key_type=app`, null `app_id`. Inventory test still passes. Run `go test -count=1 ./internal/coreapp ./internal/admin ./internal/operator -run 'TestOperatorCreateKey_|TestOperatorJSONRoutesRequirePermissionOnEachRegistration|TestParseAssigned'`.

**Verify, live.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. Ten lanes on `grok-4.6-fast-xhigh` at the PR head, per the boot recipe.

- [ ] Lane 1. Superadmin mint with default viewer role and a future expiry. Save `mint-201.png`. Pass when 201, `ak_` secret, null `app_id`.
- [ ] Lane 2. Replay GET roster. Save `mint-roster.png`. Pass when the new key is listed and the raw secret is absent.
- [ ] Lane 3. Mint with empty expiry. Save `mint-400-expiry.png`. Pass when 400 and no row.
- [ ] Lane 4. Mint with a posted `app_id`. Save `mint-platform.png`. Pass when stored `app_id` is null.
- [ ] Lane 5. Support key without `api_keys:write`. Save `mint-403.png`. Pass when 403 and no row.
- [ ] Lane 6. Viewer GUI key that can mint in GUI, JSON POST. Save `mint-viewer-json.png`. Pass when status matches `api_keys:write` on that role. If viewer lacks write, 403.
- [ ] Lane 7. Superadmin mints admin role. Save `mint-admin-role.png`. Pass when role stamps and IAM event `create_principal`.
- [ ] Lane 8. Support mints admin role. Save `mint-role-403.png`. Pass when 403 from `ParseAssignedAdminRole`.
- [ ] Lane 9. New key hits `/admin/tenants`. Save `mint-auth.png`. Pass when 200 or the role's Allows result, never 401 from a missing role.
- [ ] Lane 10. Revoke the minted key then auth. Save `mint-revoked.png`. Pass when 401 and the row remains.

**Verify, perf.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] Metric. httptest wall time of `POST /admin/operator/keys` for a superadmin key.
- [ ] Probe. Time that handler ten times at trunk (404) and at the head (201), interleaved. Also time `PUT /admin/operator/keys/:id/role` at both SHAs as a control.
- [ ] Baseline. Record the trunk role-PUT median first.
- [ ] Rule. Head mint median must not exceed five times the trunk role-PUT median.

**Review gate.** None. json-mint is not review-gated.

**Merge.**

- [ ] Root's clean verdict at the exact head SHA.
- [ ] Bugbot triage done.
- [ ] Rebased onto current trunk after the verdict, patch-id unchanged.
- [ ] The root appends json-mint after lifecycle. The operator lands the stack.

## Close the program

- [ ] Every box above is checked with its evidence.
- [ ] Reply to the operator with the report the execution playbook names.
- [ ] Deferred items still live only in `docs/specs/2026-08-30-operator-key-lifecycle/deferred.md`.

## Appendix A. Prototype evidence

No prototype ran. Open questions that a run did not settle.

JSON mint grant is `api_keys:write` to match GUI mint. Role stamp stays `admin_iam:write`. Unproven whether operators expected mint under `admin_iam` because the path sits next to `/operator/keys/:id/role`.

httptest is the live surface. Unproven against a real browser because this environment has no control-ui.

## Appendix B. Alternatives rejected

Five independent bolts on nullable `app_id`. Last-superadmin and JSON isolation still explode. Interrogate of C vs A, 2026-08-30.

Many-to-many now. No two-app hire. Additive later. Same interrogate.

Drop `api_keys_admin_app_id_null` in json-mint. CASCADE would hard-delete bound admin keys. JSON lists do not filter. All three reviewers, critical.

Mint app keys on the same POST. `/operator/` is admin principals. App keys stay GUI.

GUI delete of accounts. Disable is the leaver path. Evidence FKs SET NULL.

## Appendix C. Risks

lifecycle. `sqlc generate` and `schema.sql` in one PR. A second PR must not write those files until lifecycle merges.

lifecycle. Triggers in 023. A host without plpgsql functions named `EXECUTE FUNCTION` vs `EXECUTE PROCEDURE` (Postgres 11 vs 14). Pin the syntax that this module already uses in other migrations, or the oldest version CI runs.

json-mint. Inventory test fails a new `/operator` JSON route without `requireOp` on the same registration. Do not `Group("/operator")`.

json-mint. Raw secret in 201. Logs and error middleware must not print the body.

Live lanes. No control-ui. httptest HTML saved as the named png. Operator review of lifecycle still needs a real screen capture when a human is present.

Per-app admin keys. Starting them during this program is drift. See deferred.md.

## Appendix D. Links and reading list

Read before lifecycle. `internal/admin/gui_handler.go` `ApiKeyDelete`. `internal/queries/admin.sql` `AdminDeleteApiKey`. `internal/admin/account_repository.go` `SetDisabledAt`. `internal/operator/catalog_sql_test.go` `022` pins. `web/templates/partials/api_key_list.tmpl`.

Read before json-mint. `how` on `GenerateApiKey`, `ParseAssignedAdminRole`, `persistAPIKey`. `internal/coreapp/app.go` adminRoutes around the keys role PUT. `internal/coreapp/require_op_inventory_test.go`. `docs/specs/2026-08-22-operator-iam.md` Do not build, then ignore the JSON create ban for this program only.

lifecycle does not need interrogate. json-mint needs `how` first. Interrogate json-mint only if mint grant or secret response is contested.

Trail per `pstack/skills/show-me-your-work/SKILL.md`. Owners keep `decisions.tsv` uncommitted.

Defer list. `docs/specs/2026-08-30-operator-key-lifecycle/deferred.md`.
