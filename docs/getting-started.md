# Getting started

Two ways in: import the module into your own app, or run the reference server in this repo.

## Use it as a module

See the [project README](../README.md). You need PostgreSQL, a `core.Config`, and `app.New(cfg)`. Redis is optional (in-memory fallback). Apply the schema with `core.RunCoreMigrations` or `make migrate-up` against your database.

A working consumer lives in `examples/basic/main.go`.

## Run the reference app

This is `cmd/api`. It loads `.env` and calls `app.New()`.

### Prerequisites

- Docker and Docker Compose, or Go 1.25+, PostgreSQL 13+, Redis 6+
- Copy `.env.example` to `.env` and set at least `DB_PASSWORD` and `JWT_SECRET` (32+ characters)

### Start it

```bash
cp .env.example .env
# edit JWT_SECRET and DB_PASSWORD

make docker-dev   # Postgres, Redis, and the API with hot reload
make migrate-up   # apply schema
make setup-admin  # optional: create an admin GUI account
```

`make docker-dev` creates the Docker network it needs. The API listens on `http://localhost:8080`. Swagger is at `http://localhost:8080/swagger/index.html`. The admin GUI is at `http://localhost:8080/gui/login` unless you change `ADMIN_BASE_PATH`.

Postgres from the compose file is on host port 5433.

### App ID

Default tenant and app IDs are both `00000000-0000-0000-0000-000000000001`. They are created by the first migration.

With `MULTI_TENANT=false` (the default), callers can omit `X-App-ID` and the default app is used. With `MULTI_TENANT=true`, every non-admin request needs that header:

```bash
curl -X POST http://localhost:8080/register \
  -H "X-App-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"Pass123!@#"}'
```

Admin JSON routes use `X-Admin-API-Key`, not `X-App-ID`. OAuth callbacks carry the app ID in the OAuth state parameter.

## Day-to-day commands

```bash
make dev              # hot reload (Air), expects Postgres/Redis already up
make test             # all tests
make fmt && make lint
make security         # gosec + govulncheck
```

See the [Makefile reference](makefile-reference.md) for the rest.
