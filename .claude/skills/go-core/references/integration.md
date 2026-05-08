# Integrating go-core

How to use go-core as an auth module in your own Go application.

## Import Pattern

```go
import (
    core "github.com/MrF1ow/go-core"       // Config types, DefaultConfig, ValidateConfig
    "github.com/MrF1ow/go-core/app"          // app.New(), app.App
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

go-core embeds its migrations. Consumers apply them programmatically before starting the app:

```go
import (
    "context"
    core "github.com/MrF1ow/go-core"
    "github.com/MrF1ow/go-core/internal/database"
)

// Connect to DB
pool, err := database.ConnectPgx(host, port, user, password, dbName, sslMode)
if err != nil {
    log.Fatal(err)
}

// 1. Apply go-core schema (users, sessions, roles, etc.)
if err := core.RunCoreMigrations(context.Background(), pool); err != nil {
    log.Fatal(err)
}

// 2. Apply consumer's own migrations (app-specific tables)
if err := core.RunMigrations(context.Background(), pool, "./migrations"); err != nil {
    log.Fatal(err)
}
```

Call `RunCoreMigrations` first — it creates core tables (users, admin_accounts, roles, etc.). Then `RunMigrations` for consumer tables that reference core tables via foreign keys.

Both functions are idempotent — they track applied migrations in `schema_migrations` and skip already-applied ones.

If using the go-core repo directly for development:

```bash
make docker-dev    # starts Postgres + Redis
make migrate-up    # applies schema
```

## Protecting Consumer Routes

Use `AuthMiddleware()` on consumer route groups to require JWT auth:

```go
coreApp.RegisterRoutes(r)

// Consumer's own routes — protected by go-core JWT auth
api := r.Group("/api", coreApp.AuthMiddleware())
{
    api.GET("/orders", orderHandler.List)
    api.POST("/orders", orderHandler.Create)
}
```

JWT claims set `user_id` and `app_id` in the Gin context. Access them in consumer handlers:

```go
func (h *OrderHandler) List(c *gin.Context) {
    userID := c.GetString("user_id")
    appID := c.GetString("app_id")
    // query orders for this user...
}
```

## Admin Account Setup

Admin accounts are separate from users (stored in `admin_accounts` table). Create one via the setup tool pointed at the same DB:

```bash
# From go-core repo
make setup-admin

# Or with explicit env vars
DB_HOST=localhost DB_PORT=5433 DB_USER=postgres DB_PASSWORD=root DB_NAME=myapp DB_SSLMODE=disable go run ./cmd/setup
```

Then access the admin dashboard at `/gui/login`.

## Consumer Foreign Keys

Consumer tables should reference go-core's `users(id)` for user-owned data:

```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id UUID NOT NULL REFERENCES applications(id),
    -- consumer-specific columns
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Route Collision Avoidance

`RegisterRoutes()` mounts at root level. These paths are reserved by go-core:

- `/register`, `/login`, `/logout`, `/refresh-token`, `/forgot-password`, `/reset-password`, `/verify-email`, `/resend-verification`
- `/profile`, `/profile/*`
- `/auth/*` (OAuth callbacks)
- `/2fa/*`, `/passkey/*`, `/passkeys/*`
- `/gui/*` (admin dashboard)
- `/admin/*` (admin API)
- `/oidc/*` (OpenID Connect)
- `/health`, `/metrics`
- `/sessions`, `/sessions/*`
- `/activity-logs`, `/activity-logs/*`
- `/magic-link/*`
- `/app/*` (per-app API key routes)
- `/phone`, `/phone/*`

Consumer routes should use a distinct prefix (e.g., `/api/v1/*`).

## Common Gotchas

- **No `/api/v1` prefix** — go-core routes mount at root, not under a version prefix.
- **JWT.Secret < 32 chars** — `app.New()` returns an error. This is intentional — short secrets are a security risk.
- **Redis nil in production** — works but token blacklisting uses in-memory store. Tokens won't be invalidated across restarts or multiple instances.
- **Email nil** — magic links, email 2FA, and verification emails silently fail. Set it up if you need any of those features.
- **X-App-ID header** — all API requests need this header. Default app ID: `00000000-0000-0000-0000-000000000001`.
- **Admin GUI** — available at `/gui/login`. Admin accounts are separate from user accounts.
- **`.env` formatting** — `godotenv` silently fails on leading spaces or stray characters. Lines must start at column 0. Use `DB_SSLMODE` (not `DB_SSL_MODE`).
- **Docker dev port** — Postgres is exposed on port `5433` (not `5432`) in docker-compose.
- **Docker network** — `shared-api-network` must exist before `docker-compose up`. `dev.sh` auto-creates it.

## See Also

- `examples/basic/main.go` — runnable example
- `cmd/api/main.go` — full reference implementation with env var loading
- `references/route-map.md` — all HTTP routes
- `references/auth-flows.md` — how authentication works
