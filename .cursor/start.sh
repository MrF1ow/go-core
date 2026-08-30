#!/usr/bin/env bash
# Per-boot Postgres/Redis + schema for go-core.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/usr/local/bin:$(go env GOPATH)/bin:$PATH"

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

for _ in $(seq 1 30); do
  redis-cli ping 2>/dev/null | grep -q PONG && break
  sleep 1
done
redis-cli ping | grep -q PONG

if [ ! -f "$ROOT/.env" ]; then
  sed 's/^DB_PORT=.*/DB_PORT=5432/' "$ROOT/.env.example" > "$ROOT/.env"
fi

go run ./cmd/migrate
DB_NAME=auth_test go run ./cmd/migrate
