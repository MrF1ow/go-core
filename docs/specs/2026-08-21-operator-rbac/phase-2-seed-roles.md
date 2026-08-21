# Phase 2: Seed roles

[Overview](overview.md)

## Goal

The four system roles and the permission catalog exist in the database after migrate. Lookup by role name works.

## Changes

- Seed `operator_permissions` for every resource/action in the overview catalog.
- Seed `operator_roles`: `superadmin`, `admin`, `support`, `viewer` with `is_system=true`.
- Attach grants per the overview table. `superadmin` gets every permission row. `admin` gets every row except `admin_iam:*`.
- Idempotent `ON CONFLICT DO NOTHING`.
- `internal/operator` repository methods: get role by name, list permissions for role, list all permissions.
- Tests against a test DB or sqlmock only if the repo already uses that pattern. Prefer the same style as `internal/admin` RBAC seed tests if they exist; otherwise table-driven tests on a pure Go grant helper used by the seed so CI without Postgres still checks the intended sets.

## Data structures

Frozen map of role name to permission keys, used both by the SQL seed and by tests so the two cannot drift. **Encode Lessons in Structure:** one list, two consumers.

## Verification

Static: `go test -count=1 ./internal/operator`.

Runtime: after migrate, `GetRoleByName("viewer")` returns the four read grants only. `GetRoleByName("superadmin")` includes `admin_iam:write`.
