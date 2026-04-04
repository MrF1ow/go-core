---
name: go-core
description: Hub skill for the go-core authentication module. Routes to specific reference docs based on the task. Use this skill for ANY work in the go-core repository — it tells you which reference to read next.
---

# go-core — Authentication & Authorization Module

Multi-tenant auth module for Go. Built with Gin, PostgreSQL (pgx/SQLC), and Redis. Provides JWT auth, OAuth2 social login, WebAuthn/passkeys, 2FA, RBAC, OIDC provider, webhooks, brute-force protection, GeoIP rules, session groups, and an embedded HTMX admin GUI.

## Architecture

- **Pattern:** Repository → Service → Handler (Clean Architecture)
- **Database:** PostgreSQL via pgx connection pool + SQLC-generated queries
- **Cache/Sessions:** Redis (or in-memory fallback)
- **Public API:** `app.New(cfg)` → `RegisterRoutes(r)` → `Close()`
- **Config types:** Root package `core` (Config, DefaultConfig, CacheStore, etc.)

## When to Use Which Reference

Read the reference that matches your task. Multiple may apply.

| Task | Reference |
|------|-----------|
| Integrating go-core into a consumer app | [integration.md](references/integration.md) |
| Adding a new endpoint, feature, or domain module | [new-endpoint.md](references/new-endpoint.md) |
| Reviewing code for security issues | [security.md](references/security.md) |
| Writing a commit message | [commits.md](references/commits.md) |
| Understanding the project structure | [project-map.md](references/project-map.md) |
| Understanding the database schema | [data-model.md](references/data-model.md) |
| Understanding HTTP routes and middleware | [route-map.md](references/route-map.md) |
| Understanding auth flows and token lifecycle | [auth-flows.md](references/auth-flows.md) |
| Working on the admin GUI | [admin-gui.md](references/admin-gui.md) |
| Working on the email system | [email-system.md](references/email-system.md) |

## Quick Reference

**Entry point:** `cmd/api/main.go` (reference app) or `app.New(cfg)` (consumer)

**Key directories:**
- `app/` — public entry point (App struct, New, RegisterRoutes, Close)
- `internal/coreapp/` — internal wiring (not accessible to consumers)
- `internal/` — domain modules (user, social, twofa, webauthn, session, rbac, admin, email, log, oidc, webhook, middleware, etc.)
- `pkg/models/` — database models
- `pkg/dto/` — request/response DTOs
- `pkg/jwt/` — JWT utilities
- `pkg/errors/` — custom error types
- `web/` — embedded HTMX templates and static assets
- `migrations/` — SQL migration files

**Config:** All in root package `core` — `Config`, `DefaultConfig()`, `ValidateConfig()`

**Cross-domain wiring:** Callback function fields (not imports) to avoid circular dependencies. Set in `cmd/api/main.go` or `internal/coreapp/app.go`.
