# Phase 9: Custom role stamp

[Overview](overview.md)

PR: Custom stamp. Base is Fail-open. Exclusive `sqlc generate` writer.

## Goal

`ParseAssignedAdminRole` can stamp a non-system role that exists in `operator_roles`. A custom role with zero grants is deny, not superadmin. Non-system roles cannot carry `admin_iam`.

## Changes

- `ListOperatorRoles` and `RoleGrants` already exist. Add insert, replace-permissions, and delete-if-not-system queries. Run `sqlc generate` in this PR. No new migration unless a CHECK is the only honest way to ban `admin_iam` on non-system roles. Prefer a service reject plus a test, not a catalog rewrite.
- `ParseAssignedAdminRole`: empty post still stamps viewer. Frozen ids keep the current IAM gate. Unknown UUID is no longer `errUnknownOperatorRole` once `GetOperatorRoleByID` finds it. Missing id stays an error. Non-viewer (including any custom) still requires `admin_iam:write`.
- `OperatorAccountRole` currently rejects any id that fails `IsSystemRoleID` (`operator_handler.go`). Route account role PUT through the same stamp as keys. JSON key create does not exist. Empty PUT body stays 400, not viewer.
- Creating or updating a non-system role with an `admin_iam` permission is rejected. Seeded `admin` already omits that resource. Custom roles do not become a second superadmin.
- System names stay unwritable. `is_system` is the flag. Do not add `is_custom`.
- Zero-grant custom role: `RoleGrants` returns empty keys. `Has` denies. Auth does not fail-open.

Do not add GUI yet. JSON create/list/update/delete of custom roles may land here if they stay off `guiAuth`. Prefer JSON in this PR and GUI in phase 10 so `app.go` JSON routes and GUI routes are not one diff.

## Data structures

```
ParseAssignedAdminRole(p Principal, postedRoleID, keyType string, current *uuid.UUID) (*uuid.UUID, error)
```

Custom create takes a name, description, and a closed set of catalog `resource:action` pairs. Unknown pairs are errors. `admin_iam:*` is an error.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/admin ./internal/coreapp`.

Runtime: stamp a custom role id that exists → 204, roster role name is the custom name, `Has` matches its grants. Stamp a random UUID → error, no row change. Custom role with `admin_iam:write` in the grant list → rejected. Viewer principal stamp custom → 403.
