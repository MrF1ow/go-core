# Database migrations

Schema lives in numbered SQL files under `migrations/`. There is no ORM auto-migrate. The runner records applied files in `schema_migrations` and skips anything already done.

The same files are embedded in the module (`migrations_embed.go`). Consumers can apply them in process:

```go
pool, err := pgxpool.New(ctx, databaseURL)
if err != nil {
    return err
}
if err := core.RunCoreMigrations(ctx, pool); err != nil {
    return err
}
// then your own migrations
if err := core.RunMigrations(ctx, pool, "migrations"); err != nil {
    return err
}
```

Details and naming rules: [migrations/README.md](../migrations/README.md).

## Local Docker

Against the compose Postgres (`make docker-dev`):

```bash
make migrate-up      # pending files
make migrate-down    # last file only
make migrate-list    # files on disk
make migrate-status  # tables in the container
```

Scripts live in `scripts/`. They talk to the `auth_db` container, not to `core.RunCoreMigrations`.

## Adding a file

1. Next number: `016_short_description.sql`
2. Plain `CREATE TABLE` / `ALTER TABLE`
3. Update `internal/schema.sql` to the new final shape
4. `sqlc generate`

Do not put `IF NOT EXISTS` on brand-new tables. Seed inserts should use `ON CONFLICT DO NOTHING`. Rollback files named `_rollback.sql` or `.down.sql` are ignored by the Go runner; `make migrate-down` uses the shell rollback script.
