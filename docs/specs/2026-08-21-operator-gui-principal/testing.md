# Testing

[Overview](overview.md)

## Project commands

```
gofmt -l <touched files>   # empty
go test -count=1 ./internal/operator ./internal/middleware ./internal/admin ./internal/coreapp ./cmd/setup
```

`make test` before merge of the cutover PR that includes phases 2–4.

## Per phase

| Phase | Proof |
|-------|--------|
| 1 | Inventory test fails when a test-only `/admin` route lacks `requireOp`. Catalog test fails if a grant is removed from `016` but not `GrantsFor`, or the reverse. |
| 2 | Insert account without role fails. Viewer UUID round-trips. |
| 3 | First setup account is superadmin. Second is viewer. Overwrite does not change role. |
| 4 | Cookie session sets `KindGUIAccount`. Viewer principal lacks `tenants:write`. Missing account redirects, no panic. |

## Unavailable

No browser driver. Do not claim the sidebar or HTMX 403 works. That is the GUI shell PR.
