# Phase 1: JSON leftover levers

[Overview](overview.md)

## Goal

The two #8 review findings that get more expensive as routes land become tests, not comments.

## Changes

- Test that `operator.Catalog()` matches seeded `operator_permissions` rows and that `GrantsFor` for the four system names matches `operator_role_permissions` for the four seed UUIDs. Prefer parsing `migrations/016_operator_rbac.sql` or a shared testdata fixture over a live Postgres requirement if the package tests stay unit-level. If a DB is already available to operator tests, use it.
- Test that every Gin route under `/admin` and `GET /metrics` (and admin OIDC JSON when registered) has `RequireOperatorPermission` in its middleware chain. Walk the engine from `RegisterRoutes` with a test config. A new `/admin` handler without `requireOp` fails CI.
- Do not change null-role-on-keys behavior here. That was an explicit #8 cutover rule. GUI null-role policy is phase 4.

## Data structures

No new types. The inventory test reads Gin’s route metadata. The catalog test compares `[]Permission` to seed rows.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/coreapp`.

Runtime: add a throwaway `/admin/unguarded` in a test-only engine copy and assert the inventory test fails, then delete it. That proves the lever, not only the happy path.
