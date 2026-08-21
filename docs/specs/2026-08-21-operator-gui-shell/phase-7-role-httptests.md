# Phase 7: Role httptests

[Overview](overview.md)

## Goal

The merge bar is a test file a reviewer can rerun. Not a checklist in a PR body.

## Changes

- Add httptests that drive `RegisterRoutes` (or a GUI group with real middleware) with cookie sessions for the four system roles. Reuse the stub session and grant lookup shape from `internal/middleware/gui_auth_test.go` on main. Prefer one helper that returns an engine plus a cookie for a given role.
- Cover the merge bar in [overview.md](overview.md). Nested sessions, IAM stamp, inventory, logout, and My Account belong here if they are not already asserted in earlier phases.
- Do not add a browser suite. This environment has no control-ui. httptest HTML is the surface. Flag that in [testing.md](testing.md).

If phase 5 and 6 already contain some of these cases, this phase is the remaining gaps plus a single file a reviewer can name. Do not duplicate until the names diverge.

## Data structures

No production types. Test helper. Role in, `*gin.Engine` and session cookie out.

## Verification

Static: `go test -count=1 ./internal/coreapp ./internal/admin ./internal/middleware`.

Runtime: the merge bar, all of it, in one `go test` invocation. Then **interrogate** before marking #12 ready.
