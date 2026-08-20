# Makefile

`make help` prints this list.

## Development

| Command | Description |
|---------|-------------|
| `make setup` | Install Air, `go mod tidy` |
| `make docker-dev` | Start Postgres and Redis |
| `make docker-down` | Stop compose services |
| `make migrate-up` | `go run ./cmd/migrate` (embedded schema) |
| `make migrate-status` | List rows in `schema_migrations` |
| `make migrate-list` | SQL files in `migrations/` |
| `make setup-admin` | `go run ./cmd/setup` |
| `make dev` | Hot reload via Air (needs Postgres/Redis) |
| `make run` | `go run ./cmd/api` |
| `make build` | Binary to `bin/api` |
| `make build-prod` | Linux static binary, `CGO_ENABLED=0` |
| `make clean` | `bin/`, `tmp/`, `go clean` |
| `make install-air` | Install Air |

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

There is no `migrate-down`. Numbered schema files have no reverse SQL. Reset a local database by wiping the compose volume (`docker compose down -v`) and running `make migrate-up` again.
