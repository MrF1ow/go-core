# go-core

A multi-tenant authentication and authorization module for Go. Built on Gin, PostgreSQL (pgx/SQLC), and Redis, it handles JWT auth, OAuth2 social login, WebAuthn/passkeys, magic links, two-factor authentication, RBAC, an OIDC provider, webhooks, brute-force protection, GeoIP rules, session groups for cross-app SSO, and an embedded HTMX admin GUI. Drop it into your backend and skip building auth from scratch.

## Quick Start

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"

	core "github.com/MrF1ow/go-core"
	"github.com/MrF1ow/go-core/app"
)

func main() {
	cfg := core.DefaultConfig()
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
	coreApp.RegisterRoutes(r)
	r.Run(":8080")
}
```

That's it. You get registration, login, token refresh, password reset, email verification, 2FA, social login, and more out of the box.

## Required Config

These must be set or `app.New()` returns an error:

| Field | Description |
|-------|-------------|
| `Database.Host` | PostgreSQL host |
| `Database.Port` | PostgreSQL port (default: 5432) |
| `Database.DBName` | Database name |
| `Database.User` | Database user |
| `Database.Password` | Database password (not validated, but you need it) |
| `JWT.Secret` | Signing key for all access and refresh tokens. Minimum 32 characters. |

## Optional Config

Everything below is off or defaulted until you configure it. `DefaultConfig()` gives you sensible CORS defaults and reasonable token lifetimes.

| Field | What it does | When unset |
|-------|-------------|------------|
| `Redis` | Redis connection for token blacklisting and sessions | Nil pointer = in-memory cache. Fine for dev, use Redis in production. |
| `Email` | SMTP config for sending emails | Nil = email sending disabled. Magic links, 2FA email codes, and verification emails won't work. |
| `CORS` | Cross-origin settings | Sensible defaults via `DefaultConfig()`. Override if needed. |
| `OIDC` | OpenID Connect provider config | Disabled. |
| `WebAuthn` | Passkey and biometric authentication | Disabled. |
| `SMS` | 2FA via Twilio | Disabled. |
| `Admin` | Admin GUI settings and API key for admin endpoints | Disabled. |
| `Social` | OAuth2 social login (Google, Facebook, GitHub) | Disabled. |
| `GeoIP` | IP-based access rules, requires a MaxMind database file | Disabled. |
| `Session` | Session groups, trusted devices, cross-app SSO settings | Defaults to single-app mode. |
| `MultiTenant` | Enables multi-app mode with `X-App-ID` header | False. Single-app mode. |
| `PublicURL` | Base URL for API links in emails and redirects | Empty. |
| `FrontendURL` | Frontend app URL for redirect targets | Empty. |

## Features

- JWT authentication (access + refresh tokens)
- OAuth2 social login (Google, Facebook, GitHub)
- WebAuthn / passkeys
- Two-factor auth (TOTP, SMS, email, passkey)
- Role-based access control (RBAC)
- Multi-tenant with per-app configuration
- HTMX admin GUI (embedded, no extra files needed)
- OpenID Connect provider (auth code + PKCE)
- Webhooks
- Brute-force protection and account lockout
- GeoIP-based access rules
- Session groups (cross-app SSO)
- Activity logging

## Running the Example

Check out `examples/basic/main.go` for a working setup. You'll need PostgreSQL running with migrations applied.

```bash
# Start dependencies
make docker-dev
make migrate-up

# Run the example
go run ./examples/basic
```

`make docker-dev` spins up PostgreSQL and Redis in Docker. `make migrate-up` applies the database schema.

## Development

```bash
make dev          # Hot reload dev server
make test         # Run all tests
make lint         # golangci-lint
make build-prod   # Production binary
```

## Claude Code Skills

This project includes Claude Code skills for AI-assisted development. They live in `.claude/skills/go-core/` and cover:

- **Project map** — architecture overview and key directories
- **Route map** — all API endpoints and middleware
- **Auth flows** — registration, login, token lifecycle, 2FA, OAuth2
- **Data model** — database schema and SQLC query patterns
- **Admin GUI** — HTMX admin interface structure
- **Email system** — email templates and sending logic
- **Security** — brute-force protection, GeoIP, CSRF, rate limiting
- **New endpoint** — guide for adding new API endpoints
- **Integration** — how to consume the module in your app
- **Commits** — commit message conventions and scopes

Invoke the hub skill with `/go-core` in Claude Code to get routed to the right reference.

## Credits

Originally forked from [gjovanovicst/golang-auth-api](https://github.com/gjovanovicst/golang-auth-api). Significantly reworked into a consumable Go module — migrated from GORM to pgx/SQLC, embedded all assets, added a public API, and cleaned up the architecture.

## License

MIT.
