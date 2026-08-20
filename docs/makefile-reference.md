# Makefile

`make help` prints this list from the Makefile itself.

## Development

| Command | Description |
|---------|-------------|
| `make setup` | Install Air, `go mod tidy` |
| `make dev` | Hot reload via Air (needs Postgres/Redis) |
| `make run` | `go run ./cmd/api` |
| `make build` | Binary to `bin/api.exe` |
| `make build-prod` | Linux static binary, `CGO_ENABLED=0` |
| `make clean` | `bin/`, `tmp/`, `go clean` |
| `make install-air` | Install Air |
| `make setup-admin` | `go run ./cmd/setup` |

## Quality

| Command | Description |
|---------|-------------|
| `make test` | `go test -v ./...` |
| `make fmt` | `go fmt ./...` |
| `make lint` | golangci-lint, 5m timeout |
| `make security` | gosec + govulncheck |
| `make security-scan` | gosec only (skips `internal/sqlcgen`) |
| `make vulnerability-scan` | govulncheck |
| `make ci` | fmt, lint, test, security, build-prod |
| `make swag-init` | Regenerate `docs/` Swagger from `cmd/api/main.go` |

`make test-totp` is a leftover target. It expects `TEST_TOTP_SECRET` and `test_specific_secret.go` at the repo root. Use package tests under `internal/twofa` instead.

## Docker

| Command | Description |
|---------|-------------|
| `make docker-dev` | `./dev.sh`, compose with hot reload |
| `make docker-compose-up` | Detached production compose |
| `make docker-compose-down` | Stop compose |
| `make docker-compose-build` | Build images |
| `make docker-build` | `docker build -t auth-api .` |
| `make docker-run` | Run that image with `--env-file .env` |

## Migrations (Docker Postgres)

| Command | Description |
|---------|-------------|
| `make migrate-up` | Apply pending SQL |
| `make migrate-down` | Roll back last |
| `make migrate-list` | Files in `migrations/` |
| `make migrate-status` | `\dt` in the container |
| `make migrate-backup` | `pg_dump` into `backups/` |
| `make migrate-test` | `SELECT version()` |
| `make migrate-check` | Tables + `activity_logs` |
| `make migrate` | Interactive `scripts/migrate.sh` |
| `make migrate-init` | Create tracking table (usually already in 001) |
| `make migrate-mark-applied` | `VERSION=... NAME=...` |
