# Phase 3: Evidence export

[Overview](overview.md)

PR: Evidence export. No schema. No sqlc.

## Goal

An `admin_iam:read` principal can export access logs and IAM events as CSV, the same way it already exports the roster.

## Changes

- `GET /admin/operator/access-logs/export` and `GET /admin/operator/iam-events/export` on `admin.Handler` in `operator_handler.go`.
- `requireOp(admin_iam, read)` on `adminRoutes`. Stay on `adminRoutes` so the inventory scan sees the new lines.
- Same 10_000 cap as `operator.ExportMaxRows`. Same `X-Export-Truncated` header as roster. JSON list max is 1000. Do not raise the list default to match export.
- Filenames `operator-access-logs.csv` and `operator-iam-events.csv`. Roster CSV has no UTF-8 BOM. Match roster, not activity-log export.
- Columns match the JSON list fields. Do not add IP or UA.
- Reuse `ListAccessLogs` and `ListIAMEvents`. Do not add export SQL.

Do not log the export GET as an IAM event.

## Data structures

Reuse `AccessRecord` and `IAMEvent`. No parallel export DTO.

## Verification

Static: `go test -count=1 ./internal/admin ./internal/coreapp`.

Runtime: viewer GET export → 403 JSON. Superadmin GET → 200 CSV, cap respected, `X-Export-Truncated` set. Inventory fails if the new lines lack `requireOp`.
