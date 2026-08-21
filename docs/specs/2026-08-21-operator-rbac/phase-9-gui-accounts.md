# Phase 9: GUI accounts

[Overview](overview.md)

## Goal

Every `admin_accounts` row has an operator role. First `cmd/setup` account is `superadmin`. Further setup accounts are `viewer`. The last superadmin cannot be demoted. Roster includes people.

## Changes

- `admin_accounts.operator_role_id` NOT NULL after backfill. Existing accounts → `superadmin` (today they are gods). New accounts → `viewer`.
- Extra GUI operators are CLI-only today (`cmd/setup`). There is no `/gui/admins` route. `AccountRepository.Create` is not used by the GUI.
- `cmd/setup`: if count is 0, assign `superadmin`. Else assign `viewer`. Optional `--role` only if we need an escape hatch; default is no flag, to keep lowest privilege. Prefer no flag in v1.
- JSON create-account `POST /admin/operator/accounts` with `admin_iam:write` (username, email, password). Default role `viewer`. That is the first non-CLI way to add an operator.
- JSON `PUT /admin/operator/accounts/:id/role` with `admin_iam:write`. Refuse if it would leave zero `superadmin` accounts.
- `GUIAuthMiddleware` loads principal from the account’s role (same `OperatorPrincipal` kind `gui_account`). Do not enforce route permissions yet (phase 10). Setting principal now lets access log (phase 8) record GUI once phase 10 wires the check.
- Roster lists accounts.
- IAM events for account create and role change. Setup CLI actor kind `setup_cli`.
- Tests: last superadmin demote fails; setup first vs second account roles; middleware sets `gui_account` principal.

## Data structures

`admin_accounts.operator_role_id` FK, same as keys. Disable/leaver can be `is_disabled` if missing today. If there is no disabled flag, v1 leaver = delete account or a new `disabled_at`. Prefer `disabled_at` so IAM history still names them. Add the column in this migration if it does not exist.

## Verification

Static: `go test -count=1 ./cmd/setup ./internal/admin ./internal/middleware ./internal/operator`.

Runtime: setup tests with a test DB or repository fake. Last-superadmin unit test on the service, not only SQL.
