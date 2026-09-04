# Documentation

go-core is an importable Go module. You pass a `core.Config` to `app.New()`, mount routes, and shut it down with `Close()`. Consumers build that struct however they want. The reference app in `cmd/api` maps a `.env` file onto it. A few process settings (activity-log retention, brute-force notify flags) still read the environment through `internal/config/logging.go` and the admin settings chain.

## Using the module

| Document | Description |
|----------|-------------|
| [Quick start](../README.md) | Import, configure, and mount routes |
| [Configuration](configuration.md) | `core.Config` fields |
| [API endpoints](api-endpoints.md) | Routes and auth flows |
| [Multi-tenancy](multi-tenancy.md) | Tenants, apps, and `X-App-ID` |
| [Admin GUI](admin-gui.md) | Embedded HTMX admin panel |
| [Admin branding](../web/README.md) | Logo, colors, Custom CSS, favicon, OIDC fallback |

## Running the reference app

| Document | Description |
|----------|-------------|
| [Getting started](getting-started.md) | Local Docker setup |
| [Environment variables](guides/ENV_VARIABLES.md) | `.env` keys used by `cmd/api` |
| [OAuth redirect domains](guides/multi-app-oauth-config.md) | Allowed callback domains |

## Features

| Document | Description |
|----------|-------------|
| [Activity logging](activity-logging.md) | Event categories, anomalies, retention |
| [Operator IAM](specs/2026-08-22-operator-iam.md) | Admin JSON and GUI grants, roster, evidence, bound keys |
| [Session group expiry](session-group-expiry.md) | Cross-app logout when a session TTL expires |

## Development

| Document | Description |
|----------|-------------|
| [Architecture](ARCHITECTURE.md) | Layers, public API, data access |
| [Project structure](project-structure.md) | Directory layout |
| [Database migrations](database-migrations.md) | Schema files and how they run |
| [Testing](testing.md) | How to run tests |
| [Makefile](makefile-reference.md) | `make` targets |
| [Changelog](../CHANGELOG.md) | Release history |
| [Security policy](../SECURITY.md) | Vulnerability reporting |

## Parked

| Document | Description |
|----------|-------------|
| [Operator deferred work](specs/2026-09-03-per-app-admin-keys/deferred.md) | Last-superadmin race, many-to-many grants, and rejected IAM extras |

`pstack/` is a vendored Cursor plugin snapshot (0.14.8) so overnight plans can `git show origin/main:pstack/...`. Live `/poteto-mode` comes from the Cursor plugin. Model budget is `.cursor/rules/pstack-models.mdc`. Do not run `/setup-pstack` against the vendored defaults.

Swagger UI is served by the reference app at `/swagger/index.html`. Generated files live in this directory (`docs.go`, `swagger.json`, `swagger.yaml`) and are not hand-edited. Run `make swag-init` after changing HTTP handlers.
