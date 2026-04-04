# Integrating go-core

How to use go-core as an auth module in your own Go application.

## Import Pattern

```go
import (
    core "github.com/JedidiahDigital/go-core"       // Config types, DefaultConfig, ValidateConfig
    "github.com/JedidiahDigital/go-core/app"          // app.New(), app.App
)
```

Two packages: `core` for types, `app` for the entry point. This split exists because `internal/coreapp` imports `core` for the Config type — putting New() in the root would create a circular dependency.

## Minimal Setup

```go
cfg := core.DefaultConfig()

// Required — app.New() validates these and returns an error if missing
cfg.Database.Host = "localhost"
cfg.Database.Port = 5432
cfg.Database.DBName = "myapp"
cfg.Database.User = "postgres"
cfg.Database.Password = "secret"
cfg.JWT.Secret = "your-secret-at-least-32-characters-long"

coreApp, err := app.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer coreApp.Close()

r := gin.Default()
coreApp.RegisterRoutes(r)  // mounts all auth routes

// Add your own routes alongside
r.GET("/api/hello", func(c *gin.Context) {
    c.JSON(200, gin.H{"message": "hello"})
})

r.Run(":8080")
```

## What app.New() Does

1. Validates config (returns error for missing required fields)
2. Creates a pgx connection pool (owned by the module)
3. Initializes all internal services (user, auth, admin, OIDC, etc.)
4. Returns an opaque `*app.App` — you can't see inside it

## What RegisterRoutes() Mounts

All routes from the auth module: registration, login, token refresh, password reset, OAuth2 callbacks, 2FA endpoints, RBAC, admin GUI, OIDC provider, webhooks, health checks. See `references/route-map.md` for the full list.

## Required Config

| Field | Rule |
|-------|------|
| `Database.Host` | non-empty |
| `Database.Port` | > 0 |
| `Database.DBName` | non-empty |
| `Database.User` | non-empty |
| `JWT.Secret` | non-empty, >= 32 characters |

`Database.Password` is not validated (some local setups use empty passwords) but you'll need it for real databases.

## Optional Config

| Field | Default | Notes |
|-------|---------|-------|
| `Redis` | nil (in-memory) | Use Redis in production for token blacklisting and sessions |
| `Email` | nil (disabled) | Required for magic links, 2FA email codes, verification emails |
| `CORS` | Sensible defaults | Override `AllowedOrigins` for your domains |
| `OIDC` | Disabled | OpenID Connect provider |
| `WebAuthn` | Disabled | Passkey/biometric auth — needs RPID, RPName, RPOrigins |
| `SMS` | Disabled | 2FA via Twilio |
| `Admin` | Disabled | Set APIKey to enable admin endpoints and GUI |
| `Social` | Disabled | OAuth2 (Google, Facebook, GitHub) |
| `GeoIP` | Disabled | Needs MaxMind DB file path |
| `Session` | Single-app mode | Cross-app SSO, trusted devices |
| `MultiTenant` | false | Enables X-App-ID header routing |

## Database Migrations

go-core doesn't auto-migrate. Apply migrations before starting:

```bash
# If using the go-core repo directly
make docker-dev    # starts Postgres + Redis
make migrate-up    # applies schema

# If using as a module, run migrations from the migrations/ directory
# or use core.RunMigrations(ctx, pool, "path/to/migrations")
```

## Common Gotchas

- **JWT.Secret < 32 chars** — `app.New()` returns an error. This is intentional — short secrets are a security risk.
- **Redis nil in production** — works but token blacklisting uses in-memory store. Tokens won't be invalidated across restarts or multiple instances.
- **Email nil** — magic links, email 2FA, and verification emails silently fail. Set it up if you need any of those features.
- **X-App-ID header** — all API requests need this header. Default app ID: `00000000-0000-0000-0000-000000000001`.
- **Admin GUI** — available at `/gui/` if `Admin.APIKey` is set. Uses embedded templates — no extra files needed.

## See Also

- `examples/basic/main.go` — runnable example
- `cmd/api/main.go` — full reference implementation with env var loading
- `references/route-map.md` — all HTTP routes
- `references/auth-flows.md` — how authentication works
