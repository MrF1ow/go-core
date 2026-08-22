# Phase 8: JSON key role

[Overview](overview.md)

PR: History and accounts. Same PR as [phase 7](phase-7-iam-events.md).

## Goal

A JSON caller with `admin_iam:write` can change an admin key's role. `ParseAssignedAdminRole` is the only stamp.

## Changes

- `PUT /admin/operator/keys/:id/role` with body `{ "operator_role_id": "<uuid>" }`.
- `requireOp(admin_iam, write)`.
- Reuse `ParseAssignedAdminRole`. JSON has no API-key create handler. Do not add one.
- Same role as current is 204 and no IAM event.
- App keys 400. Unknown key 404.
- Viewer cannot PUT (403). Superadmin can.

## Data structures

No new stamp function. Request DTO is one UUID field.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/coreapp`.

Runtime: httptest assign viewer key to support as superadmin → 204, event row, roster role name support. Same PUT as viewer principal → 403, no event.
