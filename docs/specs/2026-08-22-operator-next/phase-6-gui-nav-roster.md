# Phase 6: GUI nav and roster

[Overview](overview.md)

PR: GUI roster. Read-only. CSRF not required.

## Goal

A superadmin cookie can open an Operator IAM page that lists the roster. Admin and viewer do not see the nav row. A sidebar row that 404s is not mergeable. Nav and page land in the same commit.

## Changes

- Add one System `NavSpec` row after API Keys: page `operator-iam`, path `/operator`, icon `bi-shield-shaded`, label `Operator IAM`, resource `admin_iam`, action `read`.
- Update `web/nav_test.go`. Superadmin includes the row. Admin does not. Viewer does not. `TestBuildNav_SuperadminHasEveryRowAndNoAdminIAM` becomes "has every row including Operator IAM".
- `GET /gui/operator` full page and `GET /gui/operator/roster` HTMX list. `requireGUI(admin_iam, read)`.
- Reuse `BuildRoster` / `loadRoster`. Do not add roster SQL.
- Copy the activity-log GUI shape (page + list + export), not tenant CRUD. Export can link the existing JSON CSV or a GUI wrapper of it. Same cap.
- Write CTA omit. This phase has no create/disable/role forms.

Do not add events or access-log tabs yet. That is phase 7. Empty heading rule still holds.

## Data structures

Reuse `operator.RosterEntry`. Page data is `[]RosterEntry` plus the same cap truncation flag as JSON.

## Verification

Static: `go test -count=1 ./web ./internal/admin ./internal/coreapp`.

Runtime: viewer cookie GET `/gui/operator` → 403 HTML with `X-GUI-Forbidden: 1`, no nav label Operator IAM. Superadmin GET → 200, env_key row, `requireGUI` inventory green. Admin cookie omits the nav row and GET is 403.
