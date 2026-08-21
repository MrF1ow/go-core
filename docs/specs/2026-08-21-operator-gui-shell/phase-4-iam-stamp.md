# Phase 4: IAM stamp

[Overview](overview.md)

## Goal

`api_keys:write` can mint a viewer key. It cannot stamp superadmin, admin, or support without `admin_iam:write`. Silent coerce is not the deny.

## Changes

- Add `AssignableSystemRoles` and `ParseAssignedAdminRole` in `internal/operator`. They own "who may stamp which system role on an admin API key." App keys always return a nil role. Unknown posted UUIDs stay errors, same as today's `resolveAdminOperatorRoleID`.
- Empty posted role on create stamps viewer and does not require IAM.
- Posted non-viewer role without `admin_iam:write` returns `ErrIAMAssignmentDenied`. Handlers map that to `AbortGUIForbidden`. Not a silent coerce to viewer.
- `ApiKeyUpdate` uses the same parse function. No preserve-current exception. Posting the current non-viewer role without IAM is still 403, so a name-only save cannot keep superadmin by accident of "leave it as it was."
- When the role select is hidden (phase 6), the edit form posts the current role as a hidden field so a name-only save without IAM does not demote via the empty-means-viewer create rule. The 403 is the honest outcome for non-viewer keys. Viewer keys can still be renamed.
- Point GUI create and update at the new parse. Delete `resolveAdminOperatorRoleID` once those two callers are gone. JSON has no API-key create handler. Do not invent one.

Do not add a cannot-grant-above-self lattice. `perms` stays unexported. Boolean `admin_iam:write` is the gate.

## Data structures

```
func AssignableSystemRoles(p Principal) []SystemRole
func ParseAssignedAdminRole(p Principal, postedRoleID, keyType string) (*uuid.UUID, error)
var ErrIAMAssignmentDenied error
```

`SystemRole` is the existing frozen id+name pair. Without IAM the assignable list is viewer only. With IAM it is the four system names.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/admin`.

Runtime:

- Admin principal, admin key, posted superadmin → `ErrIAMAssignmentDenied`.
- Admin principal, admin key, empty post → viewer UUID, no error.
- Superadmin principal, posted superadmin → superadmin UUID.
- Any principal, app key, posted superadmin → nil role, no error.
- Unknown UUID → error, not viewer.
- GUI handler test. Admin cookie POST `/gui/api-keys` with superadmin role is 403 HTML, no row. Omit the field, key is viewer. Superadmin cookie can stamp superadmin.
