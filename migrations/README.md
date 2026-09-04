# Migrations

SQL migrations for the go-core authentication module. These are embedded into the Go binary via `//go:embed` and applied automatically when consumers call `core.RunCoreMigrations(ctx, pool)`.

## Directory Structure

```
migrations/
├── 001_tenants_and_apps.sql    # Tenants, applications, OAuth provider configs + default tenant/app
├── 002_users.sql               # Users table
├── 003_social_accounts.sql     # Social login accounts
├── 004_activity_logs.sql       # Activity/audit logs
├── 005_admin_accounts.sql      # Admin accounts (GUI)
├── 006_api_keys.sql            # API keys + usage tracking
├── 007_system_settings.sql     # Key-value system settings
├── 008_email_system.sql        # Email configs, types, templates + seed data
├── 009_rbac.sql                # Permissions, roles, role_permissions, user_roles + seed data
├── 010_webhooks.sql            # Webhook endpoints + delivery tracking
├── 011_oidc.sql                # OIDC clients + auth codes
├── 012_trusted_devices.sql     # Trusted device tokens
├── 013_webauthn.sql            # WebAuthn/passkey credentials
├── 014_ip_rules.sql            # IP allowlist/blocklist rules
├── 015_session_groups.sql      # Cross-app SSO session groups
├── 016_operator_rbac.sql       # Operator catalog / roles
├── 017_admin_account_operator_role.sql
├── 018_operator_iam_evidence.sql
├── 019_admin_key_operator_role_required.sql
├── 020_operator_access_log_client.sql
├── 021_admin_key_must_expire.sql
├── 022_admin_account_app.sql   # Per-app GUI operators
├── 023_operator_one_way_revoke.sql
├── 024_admin_key_app_bind.sql  # Bound admin keys
└── 025_last_superadmin_guard.sql
```

## How It Works

Each migration file creates its tables in their **final state** using plain `CREATE TABLE` statements. The migration runner (`migrate.go`) tracks which files have been applied in the `schema_migrations` table and only runs pending ones.

Migrations are sorted alphabetically and run in order. The numeric prefix (`001_`, `002_`, etc.) ensures correct ordering and respects foreign key dependencies.

## For this repo

`make migrate-up` runs `go run ./cmd/migrate`, which loads `.env` and calls `RunCoreMigrations`. `make migrate-status` lists applied versions.

## For Consumers

Call `RunCoreMigrations` before your own app migrations:

```go
pool, _ := pgxpool.New(ctx, databaseURL)

// 1. Apply go-core schema (embedded)
core.RunCoreMigrations(ctx, pool)

// 2. Apply your app's migrations (from disk)
core.RunMigrations(ctx, pool, "migrations")
```

## Adding a New Migration

1. Create a new file with the next sequential number: `026_description.sql`
2. Use plain `CREATE TABLE` / `ALTER TABLE` statements (no `IF NOT EXISTS` needed for new tables)
3. Update `internal/schema.sql` to reflect the new final state (used by SQLC for code generation)
4. Run `sqlc generate` to regenerate type-safe query code

### Naming Convention

```
NNN_short_description.sql
```

- `NNN` = zero-padded sequential number
- `short_description` = snake_case summary of what the migration does

### Rules

- Each migration runs in a transaction managed by the runner
- The runner skips `_rollback.sql` and `.down.sql` files automatically
- Seed data (default tenant, email types, RBAC permissions) belongs in the same file as its table
- Use `ON CONFLICT DO NOTHING` for seed inserts to make them idempotent

## Schema Reference

`internal/schema.sql` is the consolidated final-state schema used by SQLC for code generation. It is **not** run against the database. When adding migrations, keep this file in sync with the actual schema.
