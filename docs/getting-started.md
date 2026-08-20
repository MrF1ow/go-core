# Getting started

Two ways in: import the module into your own app, or run the reference server in this repo.

## Use it as a module

See the [project README](../README.md). You need PostgreSQL, a `core.Config`, and `app.New(cfg)`. Redis is optional (in-memory fallback). Apply the schema with `core.RunCoreMigrations` or `make migrate-up`.

A working consumer lives in `examples/basic/main.go`.

## Run the reference app

`cmd/api` loads `.env` and calls `app.New()`. Postgres and Redis run in Docker. The API runs on the host.

### Prerequisites

- Docker and Docker Compose, plus Go 1.25+
- Copy `.env.example` to `.env` and set `JWT_SECRET` (32+ characters). Database defaults match compose.

### Start it

```bash
cp .env.example .env
# edit JWT_SECRET

make docker-dev    # Postgres (localhost:5433) and Redis (localhost:6379)
make migrate-up    # apply embedded schema
make setup-admin   # optional: create an admin GUI account
make dev           # cmd/api with Air hot reload
```

The API listens on `http://localhost:8080`. Swagger is at `http://localhost:8080/swagger/index.html`. The admin GUI is at `http://localhost:8080/gui/login` unless you change `ADMIN_BASE_PATH`.

`make docker-down` stops the containers.

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
make test
make fmt && make lint
make security         # gosec + govulncheck
```

See the [Makefile reference](makefile-reference.md) for the rest.
