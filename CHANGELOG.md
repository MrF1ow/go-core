# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-04

First release of go-core as an importable Go module.

Forked from [gjovanovicst/golang-auth-api](https://github.com/gjovanovicst/golang-auth-api)
and refactored from a standalone API into a self-contained library.

### Added

- **Public module API** — `app.New(cfg)`, `RegisterRoutes(r)`, `Close()` entry point
- **Config struct** — consumers build `core.Config` directly (no Viper, no env var reading)
- **CacheStore interface** — Redis or in-memory, consumer's choice
- **EmailSender interface** — pluggable email transport
- **Embedded admin GUI** — `web/` assets bundled via `embed.FS`
- **SQL migration runner** — replaces GORM AutoMigrate
- **SQLC generated queries** — type-safe database access for all repositories
- **pgx connection pool** — replaces GORM as the database layer
- **Multi-tenancy toggle** — single-tenant mode via `Config.MultiTenant = false`
- **Integration tests** — public API lifecycle tests (`app/app_integration_test.go`)
- **Example app** — `examples/basic/main.go` showing minimal consumer usage
- **CI pipeline** — test, integration test, security scan (gosec + nancy)
- **AI coding instructions** — CLAUDE.md, AGENTS.md, .cursorrules, copilot-instructions.md, opencode.json
- **Project skills** — `.claude/skills/go-core/` with endpoint, security, data model references
- **MIT License**

### Changed

- **All repositories** migrated from GORM to pgx/SQLC (user, admin, social, twofa, webauthn, webhook, rbac, oidc, email, log, bruteforce, settings, trusted devices)
- **Configuration** — replaced Viper with explicit `core.Config` struct passed by consumers
- **Composition root** — moved from `cmd/api/main.go` to `internal/coreapp/app.go`
- **Module path** — `github.com/MrF1ow/go-core`
- **CI** — removed deploy stages (library, not a service), added integration tests
- **Migration scripts** — rewritten as clean dispatchers to existing scripts
- **Documentation** — removed 50+ stale docs, updated architecture/structure/migration docs

### Removed

- **GORM** — zero GORM code remains; dependency removed from go.mod
- **Viper** — module never reads environment variables
- **GORM AutoMigrate** — replaced by SQL migration scripts
- **Deploy stages** — consumers own their deployment
- **Inherited changelog** — started fresh for v1.0.0
- **Stale documentation** — features/, implementation/, implementation_phases/, migrations/ directories
