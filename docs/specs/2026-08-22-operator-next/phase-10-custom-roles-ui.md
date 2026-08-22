# Phase 10: Custom roles UI

[Overview](overview.md)

PR: Custom roles UI. Base is Custom stamp and GUI roster.

## Goal

A superadmin cookie can create, edit, and delete non-system operator roles from the Operator IAM page, then assign them. System roles stay read-only.

## Changes

- GUI under the same `/gui/operator` nav row. A roles tab, not a second sidebar link.
- Grant editor is frozen catalog checkboxes from `ListOperatorPermissions` / `operator.Catalog()`. No free-text `resource:action`.
- `admin_iam` checkboxes are absent on the custom form. Tampered POST still rejects from phase 9.
- Delete only when `is_system` is false and no key or account still points at the role. `ON DELETE RESTRICT` already exists. Surface it as 409, not 500.
- Assign custom roles through the phase 8 forms and JSON PUT. Stamp from phase 9 is the only writer.
- Dashboard stats stay `dashboard:read`. Do not invent per-role widgets.

Do not rename seeded roles. Do not let a custom role reuse a seeded name (`idx_operator_roles_name`).

## Data structures

Reuse `SystemRole` for frozen rows. Custom rows are `operator_roles` with `is_system=false`. The editor posts a name plus a set of permission ids from the catalog.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/operator ./internal/coreapp ./web`.

Runtime: superadmin cookie creates role `auditor` with `logs:read` only → assignable, `Has("logs","read")` true, `Has("tenants","write")` false, `Has("admin_iam","write")` false. Delete while assigned → 409. Delete after reassignment → 204. Viewer GET roles tab → 403. Superadmin cannot edit the `admin` row.
