# Architecture

```
 Client  <-->  Gin routes  <-->  PostgreSQL
                    |
                    v
                  Redis (optional; in-memory fallback)
```

go-core is a library. A consumer builds `core.Config`, calls `app.New(cfg)`, and mounts routes on its own Gin engine. `cmd/api` is a reference binary that does that from environment variables.

## Public API

Package `app/`:

- `New(cfg)`: validate config, open a pgx pool, wire services
- `NewWithDB(cfg, pool)`: same, reuse an existing pool
- `RegisterRoutes(r)`: mount auth, admin GUI, OIDC, webhooks, health
- `AuthMiddleware()`: JWT middleware for the consumer's own routes
- `Close()`: stop background work and close the pool

Config types live in the root `core` package so `app` and `internal/coreapp` do not import each other in a cycle.

## Layers

Each domain under `internal/` is repository → service → handler.

- **Repository**: pgx + SQLC
- **Service**: business rules
- **Handler**: Gin binding and JSON/HTML

SQL lives in `internal/queries/`. `sqlc generate` writes `internal/sqlcgen/`. Do not edit generated files.

Cross-domain calls use function fields (`LookupRoles`, `GroupLogoutFunc`, `WebhookService`) set in `internal/coreapp/app.go`, not direct imports between domains.

## Packages

| Path | Role |
|------|------|
| `app/` | Public wrapper |
| `internal/coreapp/` | Composition root |
| `internal/user/` | Register, login, profile, magic link |
| `internal/social/` | OAuth2 + account linking |
| `internal/twofa/` | TOTP, email, SMS, backup email, trusted devices |
| `internal/webauthn/` | Passkeys |
| `internal/rbac/` | Roles and permissions |
| `internal/session/` | List/revoke sessions |
| `internal/sessiongroup/` | Cross-app SSO and expiry revocation |
| `internal/oidc/` | OIDC provider |
| `internal/webhook/` | Signed event delivery |
| `internal/bruteforce/` | Lockout and delays |
| `internal/geoip/` | MaxMind + IP rules |
| `internal/health/` | `/health`, `/metrics` |
| `internal/sms/` | Twilio |
| `internal/log/` | Activity logs |
| `internal/email/` | Templates and SMTP |
| `internal/operator/` | Admin JSON/GUI grants, catalog, bound keys |
| `internal/sso/` | Cross-app SSO token issue/exchange |
| `internal/safeconv/` | Integer conversions |
| `internal/middleware/` | JWT, CORS, rate limit, API keys, CSRF, operator grants |
| `internal/admin/` | JSON admin API + HTMX GUI |
| `pkg/jwt`, `pkg/dto`, `pkg/models`, `pkg/errors` | Shared types |

## Request path

1. Client registers or logs in (password, social, passkey, or magic link).
2. Service issues JWTs with `user_id`, `app_id`, `session_id`, `type`, and `roles`.
3. Protected routes use `Authorization: Bearer`. Redis (or memory) checks the session and blacklist.
4. RBAC, GeoIP rules, and brute-force limits run in the service or middleware.
5. Webhooks fire asynchronously with HMAC signatures.
6. OIDC clients talk to `/oidc/:app_id/...` when `OIDC.Enabled` is set.
