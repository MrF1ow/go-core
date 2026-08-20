# go-core Makefile

.PHONY: build run dev test clean setup setup-admin fmt lint ci \
	install-air install-security-tools security security-scan vulnerability-scan \
	build-prod docker-dev docker-down swag-init migrate-up migrate-status migrate-list help

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

dev:
	air

test:
	go test -v ./...

clean:
	rm -rf bin/ tmp/
	go clean

install-air:
	go install github.com/air-verse/air@latest

setup: install-air
	go mod tidy
	go mod download

setup-admin:
	go run ./cmd/setup

fmt:
	go fmt ./...

lint:
	golangci-lint run --timeout 5m

install-security-tools:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

security-scan:
	gosec -conf .gosec.json -exclude-dir=internal/sqlcgen ./...

vulnerability-scan:
	govulncheck ./...

security: security-scan vulnerability-scan

ci: fmt lint test security build-prod

build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api

docker-dev:
	docker compose up -d

docker-down:
	docker compose down

swag-init:
	swag init -g cmd/api/main.go -o docs

migrate-up:
	go run ./cmd/migrate

migrate-status:
	go run ./cmd/migrate -status

migrate-list:
	@ls -1 migrations/*.sql 2>/dev/null || echo "No migrations found"

help:
	@echo "Development:"
	@echo "  setup          Install Air and download modules"
	@echo "  docker-dev     Start Postgres and Redis"
	@echo "  docker-down    Stop compose services"
	@echo "  migrate-up     Apply embedded go-core migrations"
	@echo "  migrate-status List applied migrations"
	@echo "  migrate-list   List migration files on disk"
	@echo "  setup-admin    Create an admin GUI account"
	@echo "  run            Run cmd/api once"
	@echo "  dev            Run cmd/api with Air hot reload"
	@echo "  build          Build cmd/api to bin/api"
	@echo "  build-prod     Linux static binary"
	@echo ""
	@echo "Quality:"
	@echo "  test           go test -v ./..."
	@echo "  fmt            go fmt ./..."
	@echo "  lint           golangci-lint"
	@echo "  security       gosec + govulncheck"
	@echo "  ci             fmt, lint, test, security, build-prod"
	@echo "  swag-init      Regenerate Swagger docs"
	@echo "  clean          Remove bin/ and tmp/"
