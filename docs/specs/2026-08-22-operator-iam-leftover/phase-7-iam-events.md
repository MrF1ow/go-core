# Phase 7: IAM events

[Overview](overview.md)

PR: History and accounts. Base is Access log. Lands with [phase 8](phase-8-key-role-json.md) and [phase 9](phase-9-operator-accounts.md).

## Goal

Role assignment is an append-only event. Key create and key role change write a row.

## Changes

- Add `IAMEvent` and insert plus list on `internal/operator.Repository`.
- Hook GUI `persistAPIKey` (create_principal) and `persistAPIKeyUpdate` when `operator_role_id` changes (assign). Actor is the GUI principal on the request.
- Revoke key writes `revoke_key`. Same actor.
- `GET /admin/operator/iam-events` with `admin_iam:read`. Newest first. Optional target key or account id.
- Setup CLI write waits for phase 9 so `cmd/setup` is one commit.

Do not update or delete events.

## Data structures

```
type IAMEvent struct {
    ID uuid.UUID
    At time.Time
    ActorKind Kind // plus setup_cli as a string kind for CLI
    ActorKeyID, ActorAccountID *uuid.UUID
    TargetKind Kind
    TargetKeyID, TargetAccountID *uuid.UUID
    OldRoleID, NewRoleID *uuid.UUID
    Action string // assign, create_principal, revoke_key, disable_principal
}
```

`setup_cli` is not a request `Kind`. Store it as `actor_kind` text. Do not force it onto `Principal.Kind`.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/admin`.

Runtime: httptest GUI or JSON assign then list. Assert old and new role ids.
