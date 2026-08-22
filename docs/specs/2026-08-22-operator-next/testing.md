# Testing

[Overview](overview.md)

## Surface

No control-ui in this Cloud Agent environment. Do not claim browser verification. The matching surface is httptest through Gin JSON and HTML.

## Commands

Per phase, the packages named in that file. Before marking an impl PR ready:

```
gofmt -l $(git diff --name-only origin/main -- '*.go')
go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/coreapp ./web ./cmd/setup
```

`make ci` before the PR leaves draft.

## What proves the sequence

`Principal.Has` does not prove fail-open, CSRF HTML, or a nav row. Real routes do.

| Wave | Request | Expect |
|------|---------|--------|
| Fail-open | admin key with nil `operator_role_id` GET `/admin/x` | 401, no principal |
| Fail-open | env key GET | 200, `KindEnvKey` superadmin |
| Fail-open | `catalog_sql_test` | `019` present, CHECK, viewer backfill id |
| Evidence export | viewer GET `/admin/operator/access-logs/export` | 403 JSON |
| Evidence export | superadmin GET that path | 200 CSV, cap respected |
| Evidence retention | cleanup on a 366-day-old access row | row gone; a new row stays |
| CSRF | POST `/gui/tenants` with session, no token | 403 HTML, no `X-GUI-Forbidden` |
| CSRF | viewer GET `/gui/tenants` | 403 HTML, header present |
| GUI roster | viewer cookie GET `/gui/operator` | 403 HTML with header, nav omits Operator IAM |
| GUI roster | superadmin cookie GET `/gui/operator` | 200, `env_key` row, nav includes Operator IAM |
| GUI roster | admin cookie | nav omits Operator IAM, GET 403 |
| GUI events | superadmin GET `/gui/operator/iam-events` | 200, newest first |
| GUI mutations | last superadmin disable | 409 HTML, still enabled |
| GUI mutations | CSRF-missing POST create | 403 HTML, no header, no event |
| Custom stamp | PUT key role to a custom id that exists | 204, roster name is custom |
| Custom stamp | custom role with `admin_iam:write` | rejected |
| Custom UI | create `auditor` with `logs:read` only | assignable, no `admin_iam` |
| Custom UI | delete while assigned | 409 |

## Levers, not comments

- Migration pin in `catalog_sql_test.go` for `019`.
- Existing `require_op_inventory_test.go` on `adminRoutes`. New JSON export lines without `requireOp` fail it.
- Existing GUI inventory on `guiAuth`. New `/gui/operator` lines without `requireGUI` fail it.
- `TestCSRFForbiddenDoesNotSendGUIHeader` asserts HTML after phase 5.
- `nav_test.go` asserts Operator IAM for superadmin only.
- httptest nil-role admin key is 401. Do not keep a test that expects fail-open superadmin.

## Interrogate

Run before marking Fail-open ready (CHECK vs app keys, 401 vs 500, viewer backfill vs superadmin). Run before marking Custom-roles ready (`admin_iam` on custom roles, stamp vs `IsSystemRoleID`, delete while assigned).
