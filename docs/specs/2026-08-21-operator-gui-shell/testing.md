# Testing

[Overview](overview.md)

## Surface

No control-ui in this Cloud Agent environment. Do not claim browser verification. The matching surface is httptest through Gin, with `text/html` bodies and response headers.

JSON operator tests stay as they are. Do not weaken them.

## Commands

Per phase, the packages named in that file. Before marking #12 ready:

```
gofmt -l $(git diff --name-only origin/main -- '*.go')
go test -count=1 ./internal/middleware ./internal/admin ./internal/operator ./internal/coreapp ./web
```

`make ci` before the PR leaves draft, same as the repo rule.

## What proves the shell

Unit `Principal.Has` does not prove the GUI. A viewer cookie on a registered route does.

Minimum httptest matrix:

| Role | Request | Expect |
|------|---------|--------|
| viewer | GET `/gui/tenants` | 403, `X-GUI-Forbidden: 1`, HTML |
| superadmin | GET `/gui/tenants` | 200 |
| viewer | GET `/gui/` or `/gui/users` | 200, sidebar has Users, not Tenants, no empty Email heading |
| support | GET `/gui/users/:id/sessions` | 200 fragment |
| viewer | GET `/gui/users/:id/sessions` (HX-Request, target `#user-sessions-container`) | 403 fragment, header set |
| viewer | GET `/gui/logout` | session-only, not 403 |
| viewer | GET `/gui/my-account` | 200 |
| admin | POST `/gui/api-keys` `operator_role_id=superadmin` | 403, no row |
| admin | POST `/gui/api-keys` omit role | 201/200, role viewer |
| (inventory) | `guiAuth.GET` without `requireGUI` | test fail |

HTMX page vs fragment is already covered in `gui_permission_test.go`. Keep those tests when `AbortGUIForbidden` switches to templates.

## Levers, not comments

- Catalog pin of `NavSpec` resource strings.
- AST inventory of `guiAuth` plus `Group()` derivatives.
- Throwaway unguarded fixture in the inventory test, then delete it, so the lever is proven to fail.

## Interrogate

Run before marking #12 ready. The original principal plan called this out. Contested leftover. Nested sessions vs `users:read`, fail-closed key edit, first-paint-only nav, CSRF left as JSON.
