# Testing

[Overview](overview.md)

No control-ui. httptest through `RegisterRoutes`.

```
go test -count=1 ./internal/operator ./internal/admin ./internal/middleware ./internal/coreapp ./web ./cmd/setup
```

`make ci` before an impl PR leaves draft.

## Access client

| Who | Call | Want |
|-----|------|------|
| viewer JSON | POST `/admin/tenants` with `X-Forwarded-For` and User-Agent | 403, access row has that IP and UA |
| superadmin GUI | GET `/gui/operator/access-logs` | Table headers include IP and UA |
| superadmin JSON | GET `/admin/operator/access-logs/export` | CSV columns `ip_address`, `user_agent` |

## Must-expire

| Who | Call | Want |
|-----|------|------|
| superadmin GUI | POST admin key, empty `expires_at` | 400, no row |
| superadmin GUI | POST admin key, future datetime | Key row, non-null `expires_at` |
| superadmin GUI | POST app key, empty `expires_at` | 200, null expiry |
| any | INSERT admin key null `expires_at` | CHECK fail |
| superadmin GUI | Edit admin key, clear expiry field | Stored instant unchanged |

## Per-app

| Who | Call | Want |
|-----|------|------|
| SQL | Insert superadmin account with `app_id` set | CHECK fail |
| superadmin GUI | Create viewer bound to app A | 200, roster shows A |
| bound admin cookie | GET `/gui/tenants` | 403, `X-GUI-Forbidden: 1` |
| bound admin cookie | GET `/gui/operator` | 403 |
| bound viewer cookie | GET `/gui/users` | 200, only app A users |
| bound viewer cookie | GET `/gui/users/:id` for app B | 404 |
| platform superadmin | Demote last platform superadmin | 409 |

Existing `require_op_inventory_test.go` and GUI `requireGUI` inventory stay. A new `guiAuth.GET` without `requireGUI` still fails.
