# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Multi-tenant authentication and authorization **Go module** built with Gin, PostgreSQL (pgx/SQLC), and Redis. Consumers import `app.New(cfg)` to initialize, `RegisterRoutes(r)` to mount handlers, and `Close()` to shut down.

Features: JWT auth, OAuth2 social login, WebAuthn/passkeys, magic links, OIDC provider, RBAC, 2FA (TOTP/SMS/email/passkey), session groups (cross-app SSO), webhooks, brute-force protection, GeoIP rules, and an HTMX-based admin GUI with embedded assets.

All API requests require the `X-App-ID` header for multi-tenant context. Default app ID: `00000000-0000-0000-0000-000000000001`.

## Commands

```bash
make dev                # Hot reload cmd/api (Air); needs Postgres/Redis
make run                # Run cmd/api once
make test               # Run all tests
make fmt                # Format code
make lint               # golangci-lint (5m timeout)
make security           # gosec + govulncheck vulnerability scans
make ci                 # Run full CI pipeline (fmt, lint, test, security, build)
make swag-init          # Regenerate Swagger docs (after API changes)
make build-prod         # Cross-compile for Linux (CGO_ENABLED=0)
make docker-dev         # Start PostgreSQL + Redis via Docker
make docker-down        # Stop compose services
make migrate-up         # Apply embedded schema (cmd/migrate)
make migrate-status     # List applied migrations
make setup-admin        # Create admin account (cmd/setup)
sqlc generate           # Regenerate type-safe query code from SQL
```

### Single test execution

```bash
go test -v ./internal/user -run TestRegister              # Specific test
go test -v ./internal/user -run TestRegister -count=1     # Skip cache
go test -v -race ./internal/user                          # Race detector
go test -v -tags=integration ./app/...                    # Public API integration tests
```

## Architecture

### Public API

The module exposes a minimal surface via the `app/` package:
- `app.New(cfg)` — validates config, connects to Postgres, initializes all services
- `app.NewWithDB(cfg, pool)` — same as `New` but reuses an existing `*pgxpool.Pool`
- `app.RegisterRoutes(r)` — mounts all routes onto a Gin engine
- `app.AuthMiddleware()` — returns a `gin.HandlerFunc` for protecting consumer routes
- `app.Close()` — shuts down background services and connection pool

Configuration types live in the root `core` package (`Config`, `DatabaseConfig`, etc.). The module never reads environment variables — consumers build the `Config` struct however they want.

### Clean Architecture layers (Repository -> Service -> Handler)

Each domain module in `internal/` follows this pattern:
- `repository.go` — data access (pgx queries, SQLC generated code)
- `service.go` — business logic, orchestration
- `handler.go` — HTTP transport (Gin), request binding, response formatting

### Entry point and dependency wiring

`internal/coreapp/app.go` is the composition root: initializes all repositories, services, and handlers, then wires cross-cutting concerns (webhooks, RBAC role lookup, session groups, trusted devices) via callback fields rather than imports to avoid circular dependencies. `cmd/api/main.go` is a reference implementation that builds a `Config` from env vars and calls `app.New()`.

### Key directories

- `app/` — public entry point (`New`, `RegisterRoutes`, `Close`)
- `internal/coreapp/` — service initialization and wiring (composition root)
- `internal/` — domain modules (auth logic lives in `user/`, `social/`, `twofa/`, `webauthn/`)
- `internal/admin/` — admin GUI (HTMX templates + account/dashboard/settings services)
- `internal/middleware/` — auth JWT validation, CORS, rate limiting, API key, CSRF, IP rules
- `internal/oidc/` — OpenID Connect provider (auth code + PKCE, JWKS, introspection)
- `internal/sessiongroup/` — cross-app SSO session linking and global logout
- `internal/sqlcgen/` — SQLC generated query code (do not edit manually)
- `internal/queries/` — SQL query files for SQLC
- `pkg/models/` — database models (shared across modules)
- `pkg/dto/` — API request/response DTOs
- `pkg/jwt/` — JWT token creation and validation utilities
- `pkg/errors/` — custom error types (`NewAppError` with HTTP status codes)
- `web/` — embedded HTMX templates, static assets, and [branding](web/README.md) for admin GUI
- `migrations/` — SQL migration files (embedded; `core.RunCoreMigrations` or `make migrate-up`)
- `examples/basic/` — minimal consumer example

### Data access

All database access uses **pgx** (connection pool) and **SQLC** (generated type-safe queries). There is no ORM. SQL queries live in `internal/queries/`, generated Go code in `internal/sqlcgen/`. After changing queries, run `sqlc generate`.

### Cross-cutting patterns

- **Callback wiring**: Services expose function fields (e.g., `LookupRoles`, `GroupLogoutFunc`, `WebhookService`) set in `internal/coreapp/` to avoid circular imports between domain packages.
- **Multi-tenancy**: App-scoped via `X-App-ID` when `Config.MultiTenant` is true. Single-tenant mode injects the default app ID if the header is missing. Models reference `AppID` (UUID).
- **Configuration**: Consumers build a `core.Config` struct and pass it to `app.New()`. Some settings have a 3-tier resolution: config value > DB (admin GUI) > default.
- **Token lifecycle**: 15-min access tokens, 720-hour refresh tokens (configurable). Blacklisted via Redis on logout.
- **Email templates**: 3-tier resolution chain (DB custom > file override > embedded default). CodeMirror editor in admin GUI.

## Security

**There are NO shortcuts when it comes to security.** Every gosec finding must be properly addressed — never suppress warnings with `_ =`, `#nosec`, or blank identifiers without actually handling the issue. All errors must be explicitly checked and logged, even on "best-effort" cleanup paths. If a security scanner flags something, fix the root cause; don't silence the scanner.

## Code Style

- **Imports**: group as stdlib, third-party, internal (separated by blank lines)
- **Naming**: PascalCase exported, camelCase unexported, snake_case files, UPPER_SNAKE_CASE constants
- **Error handling**: never expose raw DB errors to clients — use `pkg/errors/` or generic messages
- **Validation**: `go-playground/validator` struct tags on DTOs
- **Swagger**: annotate all HTTP handlers; run `make swag-init` after changes

## Testing Requirements

Every new feature, bug fix, or behavioral change **must** include corresponding unit tests. Tests verify the fix works and prevent regressions. Place test files alongside the code they test (e.g., `service_test.go` next to `service.go`). Run `make test` to confirm all tests pass before considering work complete.

## Pre-commit / Pre-push Checks

Before committing or pushing, run the full CI pipeline locally:

```bash
make ci                 # Runs: fmt, lint, test, security, build-prod
```

Or individually:

```bash
make fmt                # Format code
make lint               # Lint check
make test               # Run all tests
make security           # gosec + govulncheck scans
make build-prod         # Verify production build compiles
```

All must pass before committing. Fix any failures — do not skip or bypass checks.

## Commit Messages

```
<type>(<scope>): <description>
```

Types: `feat`, `fix`, `security`, `docs`, `refactor`, `test`, `chore`
Scopes: `auth`, `user`, `social`, `twofa`, `email`, `middleware`, `database`, `redis`, `log`, `api`, `models`, `dto`, `jwt`, `webauthn`, `oidc`, `rbac`, `webhook`, `session`, `admin`

## Cloud Agents

Install and start live in `.cursor/install.sh` and `.cursor/start.sh`, wired from `.cursor/environment.json`. Postgres listens on 5432 and Redis on 6379. `make docker-dev` is not available here (no Docker, and compose publishes Postgres on 5433). Start writes `.env` from `.env.example` with `DB_PORT=5432` and applies migrations to `go_core` and `auth_test`.
