# Phase 3: Setup CLI roles

[Overview](overview.md)

## Goal

`cmd/setup` is still the only creator of GUI accounts. First account is superadmin. Every later account is viewer.

## Changes

- Count existing `admin_accounts`. Zero rows → write `operator.RoleIDSuperadmin`. Else write `operator.RoleIDViewer`.
- Password overwrite of an existing username does not change role.
- No `--role` flag in this slice.
- Print the assigned role name in the success line so the operator sees viewer vs superadmin.

## Data structures

None. Uses `RoleIDSuperadmin` / `RoleIDViewer` and `AccountRepository.Count` / `Create`.

## Verification

Static: `go test -count=1 ./cmd/setup ./internal/admin`.

Runtime: table test or testdb. Empty DB → first create has superadmin UUID. Second create has viewer UUID. Overwrite path leaves role unchanged.
