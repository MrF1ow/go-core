# Phase 6: Access log HTTP

[Overview](overview.md)

PR: Access log. Depends on [phase 5](phase-5-access-log-write.md) in the same PR.

## Goal

An `admin_iam:read` principal can list access log rows over JSON.

## Changes

- `GET /admin/operator/access-logs` on the operator JSON handler. `requireOp(admin_iam, read)`.
- Newest first. Query `limit` default 100, max 1000. Optional `decision=allow|deny`.
- Export is out of v1. Roster export is the review artifact. Access log is operational.

Viewer 403. Superadmin 200 after a fixture deny.

## Data structures

JSON list of `AccessRecord` plus `at` and `id`. Do not invent a second DTO shape.

## Verification

Static: `go test -count=1 ./internal/operator ./internal/coreapp`.

Runtime: httptest 403 as viewer. httptest deny on tenants write, then GET access-logs as superadmin. Row matches path and `deny`.
