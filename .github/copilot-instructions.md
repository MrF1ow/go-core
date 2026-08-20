# AI Coding Instructions for go-core

## Project Context & Architecture

- **Type**: Multi-tenant authentication and authorization Go module.
- **Framework**: Gin (HTTP), pgx (PostgreSQL connection pool), SQLC (generated type-safe queries), Redis (caching/sessions).
- **Public API**: Consumers import `app.New(cfg)` → `RegisterRoutes(r)` → `Close()`.
- **Structure**: Clean Architecture — Repository → Service → Handler.
  - `app/`: Public entry point (`New`, `RegisterRoutes`, `Close`).
  - `internal/coreapp/app.go`: Composition root — initializes and wires all services.
  - `internal/<domain>/`: Encapsulated features (e.g., `user`, `social`, `twofa`, `webauthn`).
  - `internal/<domain>/handler.go`: HTTP transport handling.
  - `internal/<domain>/service.go`: Business rules.
  - `internal/<domain>/repository.go`: Database interactions (pgx).
  - `internal/sqlcgen/`: SQLC generated query code (do not edit manually).
  - `internal/queries/`: SQL query files for SQLC.
  - `pkg/`: Shared code (DTOs, models, JWT utils, error types).
  - `cmd/api/`: Reference implementation — builds Config from env vars and calls `app.New()`.

## Build & Test

- **Development**: `make dev` (hot-reload via Air), `make docker-dev` (Postgres + Redis).
- **Testing**: `make test` for unit tests, `go test -v -tags=integration ./app/...` for integration tests.
- **Build**: `make build-prod` for Linux cross-compile.
- **Database**: SQL migrations in `migrations/`, applied via `make migrate-up`. No auto-migration.
- **SQLC**: After changing SQL queries in `internal/queries/`, run `sqlc generate`.
- **Swagger**: After changing handler annotations, run `make swag-init`.

## Conventions & Patterns

- **Configuration**: Consumers build a `core.Config` struct. The module never reads environment variables itself.
- **Dependency wiring**: Manual DI in `internal/coreapp/app.go`. Services use callback fields (e.g., `LookupRoles`, `AppLookup`) to avoid circular imports.
- **Validation**: `go-playground/validator` struct tags on DTOs in `pkg/dto/`.
- **Error handling**: Never expose raw database errors to API clients — use `pkg/errors/NewAppError()`.
- **Imports**: Group as stdlib, third-party, internal (separated by blank lines).
- **Naming**: PascalCase exported, camelCase unexported, snake_case files.

## Important Implementation Details

- **Multi-tenancy**: When `Config.MultiTenant` is true, API requests need `X-App-ID`. In single-tenant mode the default app ID is used if the header is missing. Models reference `AppID` (UUID).
- **Social Auth**: `internal/social/` manages OAuth flows (Google, GitHub, Facebook).
- **2FA**: `internal/twofa/` handles TOTP, SMS, email, and backup email 2FA.
- **Passkeys**: `internal/webauthn/` handles WebAuthn registration and login.
- **OIDC**: `internal/oidc/` provides OpenID Connect provider (auth code + PKCE).
- **Redis**: Used for token blacklisting, session data, and RBAC caching.
- **Admin GUI**: HTMX-based, templates embedded via `web/` package.

## Adding a New Endpoint

1. Define DTO in `pkg/dto/`.
2. Write SQL query in `internal/queries/`, run `sqlc generate`.
3. Add repository method in `internal/<domain>/repository.go`.
4. Add business logic in `internal/<domain>/service.go`.
5. Add handler in `internal/<domain>/handler.go` with Swagger annotations.
6. Register route in `internal/coreapp/app.go` `RegisterRoutes()`.
7. Run `make swag-init` to regenerate Swagger docs.
