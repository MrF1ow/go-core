# Database migrations

Schema lives in numbered SQL files under `migrations/`. There is no ORM auto-migrate. The runner records applied files in `schema_migrations` and skips anything already done.

The same files are embedded in the module (`migrations_embed.go`). Consumers apply them in process:

```go
pool, err := pgxpool.New(ctx, databaseURL)
if err != nil {
    return err
}
if err := core.RunCoreMigrations(ctx, pool); err != nil {
    return err
}
if err := core.RunMigrations(ctx, pool, "migrations"); err != nil {
    return err
}
```

`RunMigrations` is for the consumer's own SQL directory. `RunCoreMigrations` applies go-core's embedded files.

Naming rules: [migrations/README.md](../migrations/README.md).

## Local development

```bash
make docker-dev      # Postgres on localhost:5433
make migrate-up      # go run ./cmd/migrate
make migrate-status  # versions already applied
```

`cmd/migrate` reads the same `DB_*` variables as `cmd/api` (from `.env` or the environment).

There is no rollback command. To rebuild a local database: `docker compose down -v && make docker-dev && make migrate-up`.

## Adding a file

1. Next number: `025_short_description.sql`
2. Plain `CREATE TABLE` / `ALTER TABLE`
3. Update `internal/schema.sql` to the new final shape
4. `sqlc generate`

Do not put `IF NOT EXISTS` on brand-new tables. Seed inserts should use `ON CONFLICT DO NOTHING`. Files named `_rollback.sql` or `.down.sql` are ignored.
