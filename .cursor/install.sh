#!/usr/bin/env bash
# Idempotent Cloud Agent install for go-core.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export PATH="/usr/local/bin:$(go env GOPATH)/bin:$PATH"
STAGING="$(mktemp -d)"
trap 'rm -rf "$STAGING"' EXIT

go mod download

GOBIN="$STAGING" go install github.com/air-verse/air@latest
GOBIN="$STAGING" go install github.com/securego/gosec/v2/cmd/gosec@latest
GOBIN="$STAGING" go install golang.org/x/vuln/cmd/govulncheck@latest
GOBIN="$STAGING" go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b "$STAGING" v2.13.2
sudo install -m 0755 "$STAGING"/air "$STAGING"/gosec "$STAGING"/govulncheck "$STAGING"/sqlc "$STAGING"/golangci-lint /usr/local/bin/

sudo apt-get update
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  postgresql postgresql-contrib redis-server redis-tools

PG_HBA="$(ls /etc/postgresql/*/main/pg_hba.conf | head -1)"
PG_CONF="$(ls /etc/postgresql/*/main/postgresql.conf | head -1)"
sudo sed -i "s/^#\?listen_addresses.*/listen_addresses = 'localhost'/" "$PG_CONF"
sudo sed -i -E 's|^host[[:space:]]+all[[:space:]]+all[[:space:]]+127\.0\.0\.1/32[[:space:]]+.*|host all all 127.0.0.1/32 scram-sha-256|' "$PG_HBA"
sudo sed -i -E 's|^host[[:space:]]+all[[:space:]]+all[[:space:]]+::1/128[[:space:]]+.*|host all all ::1/128 scram-sha-256|' "$PG_HBA"

sudo sed -i -E 's/^#? ?supervised .*/supervised no/' /etc/redis/redis.conf
sudo sed -i -E 's/^#? ?daemonize .*/daemonize yes/' /etc/redis/redis.conf
sudo sed -i -E '/^#? ?notify-keyspace-events/d' /etc/redis/redis.conf
echo 'notify-keyspace-events Ex' | sudo tee -a /etc/redis/redis.conf >/dev/null

PG_VER="$(ls /etc/postgresql | head -1)"
if ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
  sudo pg_ctlcluster "$PG_VER" main start
fi
if ! redis-cli ping >/dev/null 2>&1; then
  sudo redis-server /etc/redis/redis.conf
fi

for _ in $(seq 1 30); do
  pg_isready -h localhost -p 5432 >/dev/null 2>&1 && break
  sleep 1
done
pg_isready -h localhost -p 5432

sudo -u postgres psql -v ON_ERROR_STOP=1 -c "ALTER USER postgres WITH PASSWORD 'postgres';"
sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='go_core'" | grep -q 1 || sudo -u postgres createdb -O postgres go_core
sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='auth_test'" | grep -q 1 || sudo -u postgres createdb -O postgres auth_test

# Pin this repo's pstack model map so skill home-path reads cannot fall back to fable/opus 5.
mkdir -p "$HOME/.cursor/rules"
cp "$ROOT/.cursor/rules/pstack-models.mdc" "$HOME/.cursor/rules/pstack-models.mdc"
