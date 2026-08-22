# Phase 5: Access log write

[Overview](overview.md)

PR: Access log. Base is Schema. Lands with [phase 6](phase-6-access-log-http.md).

## Goal

Every JSON and GUI permission decision that this leftover records can be inserted without failing the request.

## Changes

- Add `AccessRecord` and an insert method on `internal/operator.Repository`.
- Add a nil-safe `func(AccessRecord)` (or interface) field that `RequireOperatorPermission` and `RequireGUIPermission` call after the decision.
- Wire the field in `internal/coreapp`. Nil means skip. Missing principal is still 500 and does not insert.
- Insert after allow or deny. Do not insert on 500 missing principal.
- Policy: deny always. `action=write` allow always. `kind=env_key` allow always. Other read allows skip.
- Failed insert: `log.Printf`, request continues.
- Tests: 403 writes `deny`. Env allow on `tenants:write` writes `allow` with `kind=env_key`. Viewer `GET /admin/activity-logs` (read allow, not env) writes nothing.

Do not list yet. That is phase 6.

## Data structures

```
type AccessRecord struct {
    Kind Kind
    KeyID, AccountID *uuid.UUID
    RoleName, Method, Path, Decision, Resource, Action string
    Status int
}
```

`Decision` is `allow` or `deny`.

## Verification

Static: `go test -count=1 ./internal/middleware ./internal/operator`.

Runtime: httptest deny then inspect the fake insert. Do not require Postgres if the middleware test injects the func.
