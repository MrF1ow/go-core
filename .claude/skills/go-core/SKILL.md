---
name: go-core
description: Hub skill for the go-core authentication module. Routes to living docs and code. Use this skill for ANY work in the go-core repository — it tells you which reference to read next.
---

# go-core — Authentication & Authorization Module

Multi-tenant auth module for Go. Built with Gin, PostgreSQL (pgx/SQLC), and Redis. Provides JWT auth, OAuth2 social login, WebAuthn/passkeys, 2FA, RBAC, OIDC provider, webhooks, brute-force protection, GeoIP rules, session groups, operator IAM, SSO, and an embedded HTMX admin GUI.

## Architecture

- **Pattern:** Repository → Service → Handler (Clean Architecture)
- **Database:** PostgreSQL via pgx connection pool + SQLC-generated queries
- **Cache/Sessions:** Redis. `DefaultConfig()` uses `localhost:6379` DB 1. Set `cfg.Redis = nil` for in-memory.
- **Public API:** `app.New(cfg)` / `app.NewWithDB(cfg, pool)` → `RegisterRoutes(r)` → `Close()`
- **Config types:** Root package `core` (`Config`, `DefaultConfig`, `CacheStore`, etc.)
- **Composition root:** `internal/coreapp/app.go`. `cmd/api/main.go` only maps env → Config → `app.New()`.

Code is ground truth. Human docs live in `docs/`. Do not inventory file counts or line numbers here; they rot.

## When to Use Which Reference

Read the reference that matches your task. Multiple may apply.

| Task | Reference |
|------|-----------|
| Integrating go-core into a consumer app | [integration.md](references/integration.md), [docs/configuration.md](../../../docs/configuration.md) |
| Adding a new endpoint, feature, or domain module | [new-endpoint.md](references/new-endpoint.md) |
| Reviewing code for security issues | [security.md](references/security.md) |
| Writing a commit message | [commits.md](references/commits.md) |
| Project structure | [docs/project-structure.md](../../../docs/project-structure.md), [docs/ARCHITECTURE.md](../../../docs/ARCHITECTURE.md) |
| Database schema | `internal/schema.sql` and `pkg/models/` |
| HTTP routes | `internal/coreapp/app.go` `RegisterRoutes`, [docs/api-endpoints.md](../../../docs/api-endpoints.md) |
| Auth flows and token lifecycle | [auth-flows.md](references/auth-flows.md) |
| Admin GUI | [docs/admin-gui.md](../../../docs/admin-gui.md), [web/README.md](../../../web/README.md) |
| Operator IAM | [docs/specs/2026-08-22-operator-iam.md](../../../docs/specs/2026-08-22-operator-iam.md) |
| Email system | [email-system.md](references/email-system.md) |

## Quick Reference

**Entry point:** `app.New(cfg)` (consumer) or `cmd/api/main.go` (reference app)

**Key directories:**
- `app/` — public entry point (`App` struct, `New`, `NewWithDB`, `RegisterRoutes`, `Close`)
- `internal/coreapp/` — wiring (not accessible to consumers)
- `internal/` — domain modules (`user`, `social`, `twofa`, `webauthn`, `session`, `rbac`, `admin`, `operator`, `sso`, `email`, `log`, `oidc`, `webhook`, `middleware`, …)
- `pkg/models/` — database models
- `pkg/dto/` — request/response DTOs
- `pkg/jwt/` — JWT utilities
- `pkg/errors/` — custom error types
- `web/` — embedded HTMX templates, static assets, and [branding](../../../web/README.md)
- `migrations/` — SQL migration files (embedded at build time via `core.RunCoreMigrations`)

**Config:** All in root package `core` — `Config`, `DefaultConfig()`, `ValidateConfig()`

**Cross-domain wiring:** Callback function fields (not imports) to avoid circular dependencies. Set only in `internal/coreapp/app.go`.
