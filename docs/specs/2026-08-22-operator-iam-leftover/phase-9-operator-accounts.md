# Phase 9: Operator accounts

[Overview](overview.md)

PR: History and accounts. Last commit in that PR.

## Goal

JSON can create, retarget, and disable GUI operators. The last enabled superadmin cannot be demoted or disabled. A disabled account cannot use a GUI cookie.

## Changes

- `POST /admin/operator/accounts` with `admin_iam:write`. Body: username, email, password. Role is always viewer. Event `create_principal`, actor from the request principal.
- `PUT /admin/operator/accounts/:id/role` with `admin_iam:write`. Refuse with 409 `dto.ErrorResponse` if the change would leave zero enabled superadmin accounts. Event `assign` when the role actually changes.
- `POST /admin/operator/accounts/:id/disable` with `admin_iam:write`. Sets `disabled_at=now()`. Same last-superadmin 409. Event `disable_principal`. Idempotent if already disabled (204, no second event).
- Pure `WouldLeaveLastSuperadmin(enabledSuperadminCount int, targetIsEnabledSuperadmin bool) bool` in `internal/operator`. True when the target is an enabled superadmin and the enabled superadmin count is 1. Test that function, not only SQL.
- `GUIAuthMiddleware`: if `disabled_at` is set, treat as no session (redirect to login), same as a missing cookie. Do not 500.
- `cmd/setup` writes `create_principal` with `actor_kind=setup_cli` after insert. First vs extra role rules stay in `RoleIDForSetupAccount`.
- Roster JSON grows `disabled` on account rows. Env and keys omit it or send false.

Do not add delete. Disable is the leaver path. Do not add GUI `/gui/admins` in this leftover.

## Data structures

```
func WouldLeaveLastSuperadmin(enabledSuperadminCount int, targetIsEnabledSuperadmin bool) bool
```

The function returns true when `targetIsEnabledSuperadmin` is true and `enabledSuperadminCount` is 1.

Password hashing stays in `internal/admin` account create. Operator package owns the last-superadmin predicate only.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./cmd/setup`.

Runtime:

- Last superadmin demote → 409, role unchanged.
- Create account as superadmin → 201, role viewer, event row.
- Disable that viewer → 204, GUI cookie for that account redirects to login.
- Viewer principal POST create → 403.
