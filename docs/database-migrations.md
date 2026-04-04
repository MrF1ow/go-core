# Database Migrations

go-core uses SQL migrations managed by shell scripts. There is no auto-migration — all schema changes are explicit SQL files in the `migrations/` directory.

---

## Quick Commands

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check current migration status
make migrate-status

# Interactive migration tool
make migrate
```

---

## For New Contributors

```bash
# 1. Start PostgreSQL and Redis
make docker-dev

# 2. Apply all migrations (creates all tables)
make migrate-up

# 3. Start developing
make dev
```

---

## Docker Migrations

If running PostgreSQL via Docker Compose (`make docker-dev`), the migration scripts connect using the Docker container:

```bash
# Apply migrations against Docker PostgreSQL
make migrate-up

# Rollback against Docker PostgreSQL
make migrate-down
```

The connection details come from your `.env` file (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`).

---

## Creating New Migrations

1. Create a forward migration SQL file:
   ```
   migrations/YYYYMMDD_HHMMSS_description.sql
   ```

2. Create a matching rollback file:
   ```
   migrations/YYYYMMDD_HHMMSS_description_rollback.sql
   ```

3. Test and apply:
   ```bash
   make migrate-up
   ```

Migration files are applied in lexicographic order. Use the timestamp prefix to ensure correct ordering.

---

## Recent Migrations

### RBAC (Role-Based Access Control)

| Migration | Description |
|-----------|-------------|
| `20260301_add_rbac.sql` | Creates `roles`, `permissions`, and `user_roles` tables |
| `20260301_seed_rbac_defaults.sql` | Seeds default system roles (`admin`, `member`) and permissions |
| `20260302_backfill_member_role.sql` | Assigns `member` role to all existing users |

### Magic Link Login

| Migration | Description |
|-----------|-------------|
| `20260303_add_admin_magic_link.sql` | Adds `magic_link_enabled` flag to admin accounts |
| `20260303_add_magic_link_settings.sql` | Adds `magic_link_enabled` setting to applications |
| `20260303_seed_magic_link_email_type.sql` | Seeds the magic link email type into the email system |

---

## Related

- [Makefile Reference](makefile-reference.md) — All available make targets
- [Getting Started](getting-started.md) — Full setup walkthrough
