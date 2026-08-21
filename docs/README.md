# Documentation

go-core is an importable Go module. You pass a `core.Config` to `app.New()`, mount routes, and shut it down with `Close()`. The module never reads environment variables. The reference app in `cmd/api` maps a `.env` file onto that struct if you want to run the API on its own.

## Using the module

| Document | Description |
|----------|-------------|
| [Quick start](../README.md) | Import, configure, and mount routes |
| [Configuration](configuration.md) | `core.Config` fields |
| [API endpoints](api-endpoints.md) | Routes and auth flows |
| [Multi-tenancy](multi-tenancy.md) | Tenants, apps, and `X-App-ID` |
| [Admin GUI](admin-gui.md) | Embedded HTMX admin panel |
| [Admin branding](../web/README.md) | Logo, colors, and org name |

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
| [Session group expiry](session-group-expiry.md) | Cross-app logout when a session TTL expires |

## Development

| Document | Description |
|----------|-------------|
| [Architecture](ARCHITECTURE.md) | Layers, public API, data access |
| [Project structure](project-structure.md) | Directory layout |
| [Database migrations](database-migrations.md) | Schema files and how they run |
| [Testing](testing.md) | How to run tests |
| [Makefile](makefile-reference.md) | `make` targets |
| [Custom CSS spec](specs/2026-08-20-admin-custom-css-design.md) | Admin GUI extra stylesheet |
| [Favicon spec](specs/2026-08-20-admin-favicon-design.md) | Admin GUI tab icon |
| [OIDC branding spec](specs/2026-08-21-oidc-admin-branding-design.md) | Admin branding fallback on OIDC pages |
| [Changelog](../CHANGELOG.md) | Release history |
| [Security policy](../SECURITY.md) | Vulnerability reporting |

Swagger UI is served by the reference app at `/swagger/index.html`. Generated files live in this directory (`docs.go`, `swagger.json`, `swagger.yaml`) and are not hand-edited. Run `make swag-init` after changing HTTP handlers.
