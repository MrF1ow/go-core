# Phase 4: GUI principal

[Overview](overview.md)

## Goal

A valid GUI cookie request carries `*operator.Principal` with `KindGUIAccount`. Grants come from `GrantLookup.RoleGrants` on the account’s role. GUI handlers still do not check permissions.

## Changes

- `GUIAuthMiddleware` grows a `GrantLookup`, same as `AdminAuthMiddleware`. After `ValidateSession`, load `RoleGrants`. `NewPrincipal(KindGUIAccount, name, keys)`, set `AccountID`, `c.Set(OperatorPrincipalKey, &p)`. Keep the existing GUI context keys.
- Missing account (`nil, nil` from `GetByID`) is unauthenticated: clear cookie, redirect to login. Do not panic.
- Missing or unknown `operator_role_id` is 500 HTML or redirect-to-login after a logged error. Do not call `SuperadminPrincipal` for GUI. Keys keep their legacy null branch; accounts do not.
- `coreapp` passes `a.operatorRepo` into GUI auth. `GUIHandler.OperatorRepo` can stay unused until the IAM GUI exists.
- Tests in `internal/middleware/gui_auth_test.go` (new if missing). Viewer account → principal role `viewer` and `Has("tenants","write") == false`. Superadmin account → `Has("admin_iam","write")`. No `requireOp` on the test route; this phase only asserts the context key.

## Data structures

Same `operator.Principal`. `AccountID` set, `KeyID` nil. Session Redis value unchanged (admin UUID string).

## Verification

Static: `go test -count=1 ./internal/middleware ./internal/admin ./internal/coreapp`.

Runtime: httptest through `GUIAuthMiddleware` with a fake session validator and fake `GrantLookup`. Assert `OperatorPrincipalKey`. Deleted account → 302 to login, process lives.
