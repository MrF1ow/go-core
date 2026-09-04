# Last-superadmin race plan

A dedicated Postgres trigger stops concurrent disable or demote from clearing the last platform superadmin. Operators keep one enabled platform GUI superadmin. The Go 409 check stays. PR-1 is the only PR.

## How to read this

One box is one unit of work. Every box names the evidence that checks it. A nested box is a sub-step of the box above it. Check a box only when its evidence exists, a file, a log line, a screenshot, a test run, or a SHA. The body is a how-to. The appendices explain and record.

The program runs `pstack/skills/poteto-mode/playbooks/feature.md`. The operator merges PR-1 after it is merge-ready. The owner stops at merge-ready.

Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

## Program checklist

### Arm the program

- [ ] State the protocol and this plan to the operator, then stop. Start execution only on her explicit go.
- [ ] On her go, arm a `/goal` with this exact text. "docs/specs/2026-09-04-last-superadmin-race/plan.md. PR-1. Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. The operator merges. Done when one enabled platform superadmin remains after concurrent disable or demote, and deferred.md no longer lists this race."
- [ ] Read these from trunk at program start. Re-read them at every tick.
  - [ ] `git show origin/main:pstack/skills/poteto-mode/playbooks/feature.md`
  - [ ] `git show origin/main:pstack/skills/swarm/SKILL.md`
  - [ ] `git show origin/main:docs/admin-gui.md`
  - [ ] `git show origin/main:pstack/skills/poteto-mode/playbooks/opening-a-pr.md`
  - [ ] `git show origin/main:pstack/skills/principle-prove-it-works/SKILL.md`
  - [ ] `git show origin/main:pstack/skills/principle-sequence-verifiable-units/SKILL.md`
  - [ ] `git show origin/main:docs/specs/2026-09-03-per-app-admin-keys/deferred.md`
- [ ] Arm the 30-minute audit tick. In a local session, a real terminal `/loop`. In a cloud root, a cloud-sleeper wake chain. Never leave the cadence to memory.
- [ ] Use this tick prompt, verbatim. "Re-read the execution playbook from trunk and the armed /goal. Audit the operation against both and fix drift in this tick. Probe every active lane and judge progress by side effects only. Stand down a stuck lane and dispatch its replacement now. Then send the operator a status message, whether or not anything changed, with the queue table of PR, owner, state, and head SHA, the verdicts since the last tick, what merged, open operator gates, and blockers."
- [ ] On the operator's hold or stand-down, send every owner a zero-writes order at once.

### Spawn owners

- [ ] Spawn one owner per PR with the full lifecycle the execution playbook names.
- [ ] Follow this dependency graph. Start dependent work only after its parent merges, or base it on the parent branch when the execution playbook stacks.
  - [ ] PR-1 is the only PR. It branches from `main`.
- [ ] Hold the file boundaries. PR-1 touches only the files listed in its Files block.
- [ ] Hold the review gate. PR-1 changes no interaction. It is not review-gated.

### PR mechanics, for every PR

- [ ] Resolve the forge once. Default to `gh`; if `command -v origin` succeeds and Origin can resolve the repository, use `origin pr` for every PR operation. Record any fallback to `gh`. Never require `gt`.
- [ ] Open the PR ready, never draft, with `origin pr create --status open --base main` or `gh pr create --base main` according to the resolved forge. A stack child targets its parent branch.
- [ ] Run the repo's lint and typecheck once before the PR-facing push. Push with hooks on.
- [ ] Run `/deslop` before each commit and `/no-comments` before review.
- [ ] Triage every Bugbot and security-reviewer comment per `pstack/skills/poteto-mode/references/bugbot-triage.md`.
- [ ] Rebase onto current trunk before babysit and again before the merge-ready report.

### Verdict and merge, for every PR

- [ ] At the merge-ready head SHA, run the swarm per `pstack/skills/swarm/SKILL.md`. One gates lane. The ten live lanes from the PR's **Verify, live** block. The perf lane from its **Verify, perf** block. One audit lane that reads the diff and the receipts and distrusts the PR body.
- [ ] Clean only when every lane is `PASS`. Findings go back to the owner. A new head gets a fresh swarm and a fresh verdict.
- [ ] Record the verdict head SHA, base SHA, and stable `git patch-id` of the base-to-head diff. The operator squash-merges PR-1. Re-verify when the patch-id changes.

### Boot recipe, for every live lane

Each live lane runs on its own cloud VM at the PR head. Drive JSON with `curl`. Drive GUI pages with the same HTTP session the admin GUI uses. `control-ui` is not in this repo.

- [ ] `git fetch origin <head-branch> && git checkout <head SHA>`.
- [ ] Start Postgres on 5432 and Redis on 6379. Apply migrations with `go run ./cmd/migrate`. Start `go run ./cmd/api`. Wait until `GET /health` or the process listen log shows ready.
- [ ] Deliver JSON through `curl` with `X-App-ID` and the admin key. Deliver GUI through `curl` cookie plus CSRF as `docs/admin-gui.md` describes. Read-only diagnostics are `psql` counts on `admin_accounts` and the API JSON body.
- [ ] Save every screenshot to `/tmp/swarm-pr-1/worker-<n>/<slug>.png` and return the paths with the report.

## Close the last-superadmin race (PR-1)

**Depends on.** None.

**Files.**

- [ ] Create `migrations/025_last_superadmin_guard.sql`.
- [ ] Edit `internal/schema.sql`.
- [ ] Edit `internal/admin/account_repository.go`.
- [ ] Edit `internal/admin/operator_handler.go`.
- [ ] Edit `internal/admin/gui_iam.go`.
- [ ] Edit `internal/operator/catalog_sql_test.go`.
- [ ] Edit `internal/coreapp/operator_iam_test.go`.
- [ ] Edit `docs/specs/2026-09-03-per-app-admin-keys/deferred.md`.
- [ ] Edit `docs/specs/2026-08-22-operator-iam.md`.
- [ ] Edit `docs/README.md`.
- [ ] Edit `migrations/README.md`.

**Build.**

- [ ] Add `prevent_last_superadmin_cleared` as a `BEFORE UPDATE` trigger on `admin_accounts` in the 023 style. Take `pg_advisory_xact_lock(hashtextextended('admin_accounts.last_enabled_superadmin', 0))` before the count. Raise `cannot demote or disable the last enabled superadmin` when the update would leave zero enabled platform superadmins (`operator_role_id = 'd0000000-0000-0000-0000-000000000001'`, `disabled_at IS NULL`, `app_id IS NULL`, `id <> NEW.id`). Do not wrap the four handlers in transactions. Keep the Go `WouldLeaveLastSuperadmin` 409. Map the trigger error to 409 in `AccountRepository.SetDisabledAt` and `AccountRepository.UpdateOperatorRole`, then check that error in JSON and GUI disable and role-change writes. Copy the function and trigger into `internal/schema.sql`. Remove the last-superadmin section from deferred.md and point the IAM recap at this plan. The SQL to land is this shape.

```sql
CREATE OR REPLACE FUNCTION prevent_last_superadmin_cleared()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    remaining int;
BEGIN
    IF (OLD.operator_role_id = 'd0000000-0000-0000-0000-000000000001'::uuid
        AND OLD.disabled_at IS NULL
        AND OLD.app_id IS NULL)
       AND NOT (NEW.operator_role_id = 'd0000000-0000-0000-0000-000000000001'::uuid
        AND NEW.disabled_at IS NULL
        AND NEW.app_id IS NULL)
    THEN
        PERFORM pg_advisory_xact_lock(
            hashtextextended('admin_accounts.last_enabled_superadmin', 0)
        );
        SELECT COUNT(*) INTO remaining FROM admin_accounts
        WHERE id <> NEW.id
          AND operator_role_id = 'd0000000-0000-0000-0000-000000000001'::uuid
          AND disabled_at IS NULL
          AND app_id IS NULL;
        IF remaining = 0 THEN
            RAISE EXCEPTION 'cannot demote or disable the last enabled superadmin';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER admin_accounts_last_superadmin
    BEFORE UPDATE ON admin_accounts
    FOR EACH ROW
    EXECUTE FUNCTION prevent_last_superadmin_cleared();
```

- [ ] Do not add duration presets, GUI delete, many-to-many grants, or JSON mint of app-type keys.

**You see.**

- [ ] Two concurrent disables of two enabled platform superadmins leave exactly one enabled. The blocked write returns HTTP 409 with `cannot demote or disable the last enabled superadmin` and writes no IAM event.
- [ ] Sequential disable of the last remaining superadmin still returns 409 before the UPDATE, same JSON and HTML as today.

**Verify, unit.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] `internal/operator/catalog_sql_test.go` gains a 025 needle test and a live two-session race against `auth_test`. Run `go test -v ./internal/operator -run 'TestLastSuperadmin|TestOperatorOneWayRevokeMigration' -count=1`.
- [ ] `internal/coreapp/operator_iam_test.go` still passes last-superadmin 409 and maps a `P0001` write error to 409 with no IAM event. Run `go test -v ./internal/coreapp -run 'TestOperatorDisableAccount_LastSuperadmin|TestOperatorAccountRole_LastSuperadmin|TestGUIShell_LastSuperadmin' -count=1`.

**Verify, live.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked. Ten lanes on the swarm workers model at the PR head, per the boot recipe.

- [ ] Lane 1. Regression lane against trunk. Run two concurrent JSON disables of two enabled platform superadmins at trunk and head. Trunk lacks the trigger, so record that and gate head leaving one enabled superadmin plus one 409. Save `regression-concurrent-disable.png`. Pass when trunk can reach zero enabled platform superadmins and head cannot.
- [ ] Lane 2. Two concurrent JSON demotes of two enabled platform superadmins to admin. Save `concurrent-demote.png`. Pass when one role stays superadmin and the other request is 409.
- [ ] Lane 3. Concurrent JSON disable of superadmin A and demote of superadmin B. Save `mixed-disable-demote.png`. Pass when one enabled platform superadmin remains.
- [ ] Lane 4. Sequential JSON disable of the last remaining superadmin. Save `sequential-last-disable.png`. Pass when status is 409, `disabled_at` stays null, and no IAM event row is inserted.
- [ ] Lane 5. JSON disable of a viewer while two superadmins exist. Save `viewer-disable.png`. Pass when status is 204 and the superadmin count stays 2.
- [ ] Lane 6. Disable one of two superadmins, then disable the last. Save `second-disable-409.png`. Pass when the first is 204 and the second is 409.
- [ ] Lane 7. POST that would clear `disabled_at` on an already disabled account. Save `reenable-blocked.png`. Pass when 023 still raises `disabled_at cannot be cleared`.
- [ ] Lane 8. GUI POST `/gui/operator/accounts/:id/disable` on the last enabled superadmin. Save `gui-last-disable-409.png`. Pass when the response is HTML 409 and contains `cannot demote or disable the last enabled superadmin`.
- [ ] Lane 9. Disable a bound non-superadmin GUI account. Save `bound-non-super-disable.png`. Pass when status is 204 and platform superadmin count is unchanged.
- [ ] Lane 10. Concurrent JSON disable of the last superadmin and a viewer. Save `last-plus-viewer.png`. Pass when the viewer is 204, the last superadmin is 409, and one enabled platform superadmin remains.

**Verify, perf.** Tests alone are not sufficient verification. A PR is verified only when its unit, live, and perf boxes are all checked.

- [ ] Metric. Median wall time of 50 sequential JSON disables of a viewer account at trunk and at head. Also median wall time of 50 sequential disables of one of two platform superadmins at head only, because trunk can clear the last superadmin and that path is unlike.
- [ ] Probe. `curl` `POST /admin/operator/accounts/:id/disable` in a loop, interleaved trunk then head for the viewer path. Recreate a fresh viewer between iterations. Both sides must print the median in milliseconds.
- [ ] Baseline. Record the trunk viewer-disable median first.
- [ ] Rule. Head viewer-disable median fails when it exceeds twice the trunk median. Head superadmin-disable median fails when it exceeds 100ms. Do not ratio the superadmin path against trunk.

**Review gate.** None. PR-1 is not review-gated.

**Merge.**

- [ ] Root's clean verdict at the exact head SHA.
- [ ] Bugbot triage done.
- [ ] Rebased onto current trunk after the verdict, patch-id unchanged.
- [ ] The operator squash-merges PR-1.

## Close the program

- [ ] Every box above is checked with its evidence.
- [ ] Reply to the operator with what PR-1 changed, the live lane table, and the merge SHA.

## Appendix A. Prototype evidence

Question. Does a 023-style `BEFORE UPDATE` count close two concurrent disables of the last two platform superadmins?

Throwaway program. `/tmp/last-superadmin-proto/main.go` against `auth_test` on localhost 5432. 40 rounds each. Copy of results in `/opt/cursor/artifacts/last-superadmin-proto-results.txt`.

| variant | zero superadmins | raises | deadlocks |
| naive_count | 39 | 1 | 0 |
| select_for_update | 0 | 27 | 13 |
| lock_table | 0 | 22 | 18 |
| advisory_xact_lock | 0 | 40 | 0 |
| advisory_mixed_disable_demote | 0 | 40 | 0 |

Naive count loses. `SELECT FOR UPDATE` and `LOCK TABLE` keep one superadmin and deadlock. `pg_advisory_xact_lock` keeps one superadmin and never deadlocked, including mixed disable plus demote.

Unproven. A `BEFORE DELETE` path. `DeleteByID` has no HTTP caller. Out of this PR.

## Appendix B. Alternatives rejected

Wrap the four handlers in a transaction. Deferred.md forbids it. The race is in Postgres, not in Gin.

Naive 023 count with no lock. Prototype left zero superadmins in 39 of 40 rounds.

`SELECT FOR UPDATE` of enabled platform superadmins. Holds row locks the sibling UPDATE already needs. Deadlock in 13 of 40 rounds. HTTP would see `40P01`.

`LOCK TABLE ... SHARE ROW EXCLUSIVE` inside the same `BEFORE UPDATE`. Conflicts with the table lock the UPDATE already holds. Deadlock in 18 of 40 rounds.

Hitchhike onto a mint PR. Deferred.md forbids it. Disable and demote did not change in json-scope or bind.

Remove the Go `WouldLeaveLastSuperadmin` check. Sequential 409 tests and IAM-event-on-block tests already use it. Keep it. The trigger is the race net.

App-type keys as operator principals. Deferred never.

## Appendix C. Risks

PR-1. `control-ui` and `control-cli` are not in this repository. Live lanes use `curl`. GUI lane 8 uses an HTTP session, not a browser control skill.

PR-1. The advisory lock serializes only updates that leave enabled-platform-superadmin. Viewer disables skip the lock. Watch the superadmin-disable median in Verify, perf.

PR-1. If `AccountRepository` mapping misses `UpdateOperatorRole`, concurrent demote returns 500. The mixed live lane and the role-change unit test catch that.

PR-1. `DeleteByID` can still remove a row without the UPDATE trigger. No HTTP caller. Do not add a delete button. Leave the unused query for the next `admin_account.sql` PR named in deferred.md.

## Appendix D. Links and reading list

Read before editing.

- `docs/specs/2026-09-03-per-app-admin-keys/deferred.md`
- `docs/specs/2026-08-22-operator-iam.md`
- `migrations/023_operator_one_way_revoke.sql`
- `internal/queries/admin_account.sql` (`CountEnabledSuperadminAccounts`)
- `internal/operator/iam_event.go` (`WouldLeaveLastSuperadmin`)
- `internal/admin/operator_handler.go` (`OperatorDisableAccount`, role PUT)
- `internal/admin/gui_iam.go` (`OperatorDisableAccount`, role POST)
- `internal/admin/account_repository.go` (`SetDisabledAt`, `UpdateOperatorRole`)
- `internal/operator/catalog_sql_test.go` (`TestAdminKeyAppBindLive`, `bindTestPool`)
- `docs/database-migrations.md`

PR-1 runs `pstack/skills/how/SKILL.md` only if the trigger shape in Build no longer matches Postgres. Skip `pstack/skills/architect/SKILL.md`. The prototype already compared four shapes. Skip `pstack/skills/interrogate/SKILL.md` unless review contests the advisory lock. No `pstack/skills/show-me-your-work/SKILL.md` trail unless the operator asks for one.
