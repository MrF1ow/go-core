# Testing

[Overview](overview.md)

## Surface

No control-ui in this Cloud Agent environment. Do not claim browser verification. The matching surface is httptest through Gin JSON (and HTML only for the disabled-account login redirect).

## Commands

Per phase, the packages named in that file. Before marking an impl PR ready:

```
gofmt -l $(git diff --name-only origin/main -- '*.go')
go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/coreapp ./cmd/setup
```

`make ci` before the PR leaves draft.

## What proves leftover IAM

`Principal.Has` does not prove roster or last-superadmin. Real routes do.

| Wave | Request | Expect |
|------|---------|--------|
| Roster | viewer GET `/admin/operator/roster` | 403 JSON |
| Roster | superadmin GET `/admin/operator/roster` | 200, `kind=env_key`, at least one role name |
| Roster | superadmin GET `/admin/operator/roster/export` | 200 CSV, cap respected |
| Schema | `catalog_sql_test` | `018` present, `disabled_at` on `admin_accounts` |
| Access log | viewer POST `/admin/tenants` | 403, access row `deny` |
| Access log | env key POST `/admin/tenants` (or equivalent write) | 200 or 201, access row `allow` `kind=env_key` |
| Access log | viewer GET `/admin/activity-logs` | 200, no access row for that read allow |
| History | superadmin PUT `/admin/operator/keys/:id/role` | 204, iam-events old and new ids |
| Accounts | last superadmin PUT role to admin | 409, no event |
| Accounts | superadmin POST `/admin/operator/accounts` | 201 viewer, event `create_principal` |
| Accounts | disabled GUI cookie GET `/gui/users` | redirect login, not 500 |

## Levers, not comments

- Migration pin in `catalog_sql_test.go` for `018`.
- Existing `require_op_inventory_test.go` on `adminRoutes`. A new operator JSON line without `requireOp` fails that test.
- `WouldLeaveLastSuperadmin` unit tests covering count 0, 1, 2.
- Fake access-log func in middleware tests so deny/allow policy is proven without Postgres.

## Interrogate

Run before marking Schema ready (migration vs `schema.sql` drift, sqlc exclusive writer). Run before marking History-and-accounts ready (last-superadmin, disable vs delete, setup_cli actor, GUI disabled redirect).
