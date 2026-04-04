# Documentation

Complete documentation for the go-core authentication module.

---

## Using go-core as a Module

| Document | Description |
|----------|-------------|
| [Quick Start](../README.md) | Import, configure, and mount routes in 10 lines |
| [Configuration](configuration.md) | Environment variables, OAuth, and logging config |
| [API Endpoints](api-endpoints.md) | Full endpoint reference and authentication flows |
| [API Reference (detailed)](API.md) | Complete request/response schemas and examples |

The public API surface is `app.New(cfg)` → `RegisterRoutes(r)` → `Close()`. Config types live in the root `core` package. See the [project README](../README.md) for a minimal working example.

---

## Getting Started

| Document | Description |
|----------|-------------|
| [Getting Started](getting-started.md) | Installation, prerequisites, and first steps |
| [Configuration](configuration.md) | Environment variables, OAuth, and logging config |
| [Admin GUI](admin-gui.md) | Setting up and using the admin panel |

---

## Using the API

| Document | Description |
|----------|-------------|
| [API Endpoints](api-endpoints.md) | Full endpoint reference and authentication flows |
| [API Reference (detailed)](API.md) | Complete request/response schemas and examples |
| [Multi-Tenancy](multi-tenancy.md) | Tenant/app management, data isolation, OAuth per-app |
| [Activity Logging](activity-logging.md) | Smart logging, anomaly detection, retention policies |
| [Swagger UI](http://localhost:8080/swagger/index.html) | Interactive API docs (when running) |

---

## Development

| Document | Description |
|----------|-------------|
| [Project Structure](project-structure.md) | Codebase layout and key files |
| [Architecture](ARCHITECTURE.md) | System design, patterns, and layers |
| [Database Migrations](database-migrations.md) | Migration system and commands |
| [Testing](testing.md) | Running tests, coverage, and pre-commit checks |
| [Makefile Reference](makefile-reference.md) | All available make commands |
| [Contributing](../CONTRIBUTING.md) | Contribution process and standards |
| [Code of Conduct](../CODE_OF_CONDUCT.md) | Community guidelines |

---

## Setup Guides

- [Environment Variables](guides/ENV_VARIABLES.md) — Complete environment variable reference
- [Validation Endpoint](guides/auth-api-validation-endpoint.md) — Auth validation endpoint guide
- [Multi-App OAuth Config](guides/multi-app-oauth-config.md) — Per-application OAuth setup
- [Nancy Setup](guides/NANCY_SETUP.md) — Dependency vulnerability scanner setup

---

## Reference

- [Pre-Release Migration Guide](BREAKING_CHANGES.md) — For early fork users upgrading
- [Changelog](../CHANGELOG.md) — Version history and release notes
- [Security Policy](../SECURITY.md) — Vulnerability reporting

---

## Quick Lookup

| I want to... | Go to |
|--------------|-------|
| Use go-core in my app | [Quick Start](../README.md) |
| Set up the project | [Getting Started](getting-started.md) |
| Configure environment | [Configuration](configuration.md) |
| See API endpoints | [API Endpoints](api-endpoints.md) |
| Set up social login | [Configuration - OAuth](configuration.md#social-authentication) |
| Set up passkeys/WebAuthn | [Configuration - WebAuthn](configuration.md#webauthn--passkeys) |
| Set up magic link login | [API Endpoints - Magic Link](api-endpoints.md#magic-link-login) |
| Manage roles & permissions | [API Endpoints - RBAC](api-endpoints.md#rbac-administration) |
| Manage sessions | [API Endpoints - Sessions](api-endpoints.md#session-management) |
| Link social accounts | [API Endpoints - Social Linking](api-endpoints.md#social-account-linking) |
| Manage tenants/apps | [Multi-Tenancy](multi-tenancy.md) |
| Use the admin panel | [Admin GUI](admin-gui.md) |
| Run database migrations | [Database Migrations](database-migrations.md) |
| Run tests | [Testing](testing.md) |
| Contribute code | [Contributing](../CONTRIBUTING.md) |
| Understand the architecture | [Architecture](ARCHITECTURE.md) |
| See all make commands | [Makefile Reference](makefile-reference.md) |
| Configure logging | [Activity Logging](activity-logging.md) |
