# Phase 1: Roster types

[Overview](overview.md)

PR: Roster. Lands with [phase 2](phase-2-roster-http.md).

## Goal

A pure assembler turns admin keys, GUI accounts, and one synthetic env-key row into `[]RosterEntry`. HTTP is phase 2.

## Changes

- Add `RosterEntry` and `BuildRoster` in `internal/operator`.
- Map frozen role IDs to names in Go. Do not JOIN `operator_roles` in new SQL. `AdminListApiKeys` already returns `operator_role_name` for keys. Accounts come from `ListAllAdminAccounts` with `operator_role_id` only.
- Env-key row is always present. `Kind=env_key`. No `KeyID`. No `AccountID`. `RoleName=superadmin`. `ExpiresAt` nil. `Revoked=false`. `DisplayName` is a fixed `env_key` string, not the secret.
- Null `expires_at` stays nil (forever). Expired keys stay on the roster.
- App-type keys are omitted. Roster is operator principals.
- A null `api_keys.operator_role_id` shows an empty role name. That is the fail-open leftover, not a JOIN bug.
- Cap assembly at `ExportMaxRows` (10_000), same number as `internal/log.ExportMaxRows`. Duplicate the const in `internal/operator`. Do not import `internal/log`.
- `Disabled` is omitted in v1. The accounts PR adds it.

Do not log roster assembly as IAM history.

## Data structures

```
type RosterEntry struct {
    Kind, DisplayName, RoleName string
    KeyID, AccountID *uuid.UUID
    CreatedAt time.Time
    LastUsedAt, ExpiresAt *time.Time
    Revoked bool
}

func BuildRoster(env RosterEntry, keys []RosterEntry, accounts []RosterEntry) []RosterEntry
```

Env row is supplied by the caller so tests do not read config. `BuildRoster` prepends env, then keys, then accounts, then truncates to the cap.

## Verification

Static: `go test -count=1 ./internal/operator`.

Runtime: none. Phase 2 is HTTP.
