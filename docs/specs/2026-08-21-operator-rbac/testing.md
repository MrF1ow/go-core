# Testing

[Overview](overview.md)

## Project commands

```
make fmt
go test -count=1 ./internal/operator ./internal/middleware ./internal/admin ./internal/coreapp ./cmd/setup ./web
make test
make lint
make security
```

Cutover PRs (phases 4 and 10) run `make ci` before merge.

## Surfaces

| Phase | Surface | How |
|-------|---------|-----|
| 1–2 | Postgres migrate | `make migrate-up` or CI `RunCoreMigrations` |
| 3–8 | JSON | httptest + Gin, `X-Admin-API-Key` |
| 9 | CLI | `cmd/setup` tests |
| 10 | GUI | template render + httptest cookie session |

No Chrome / control-ui in this cloud environment. Do not add screenshot tests.

## Fixtures

Reuse one helper: `principal(roleName)` that attaches an `OperatorPrincipal` without hashing keys, for middleware unit tests. Separate fixture: hashed DB key with `operator_role_id` for handler tests that run `AdminAuthMiddleware` for real.

## Regression

Keep existing `HasScope` tests. They must not be used by `/admin/*` after phase 4. Add a test that admin auth middleware does not set `ApiKeyScopesKey` for admin keys (or that `RequireOperatorPermission` ignores it).

There is no `admin_auth_test.go`. Add one in phase 3 covering env vs DB principal attachment. Fix `app_api_key_integration_test.go` middleware arity in phase 4.
