# Project structure

```
.
├── app/                        # Public API: New, RegisterRoutes, Close
├── cmd/
│   ├── api/                    # Reference server (loads .env)
│   ├── migrate_oauth/          # One-shot: env OAuth creds → database
│   └── setup/                  # Admin GUI account wizard
├── examples/basic/             # Minimal consumer
├── internal/
│   ├── coreapp/                # Wiring
│   ├── admin/                  # Admin JSON API + HTMX GUI
│   ├── user/                   # Users, magic link, import/export
│   ├── social/                 # OAuth2
│   ├── twofa/                  # TOTP, email, SMS, backup email, trusted devices
│   ├── webauthn/               # Passkeys
│   ├── rbac/
│   ├── session/
│   ├── sessiongroup/           # Cross-app SSO
│   ├── oidc/
│   ├── webhook/
│   ├── bruteforce/
│   ├── geoip/
│   ├── health/
│   ├── sms/
│   ├── log/
│   ├── email/
│   ├── middleware/
│   ├── database/               # pgx pool helpers
│   ├── redis/
│   ├── settings/
│   ├── config/                 # Activity-log config (still reads env)
│   ├── queries/                # SQLC source
│   ├── sqlcgen/                # Generated query code
│   ├── schema.sql              # SQLC schema snapshot (not applied)
│   └── util/
├── pkg/
│   ├── models/
│   ├── dto/
│   ├── errors/
│   └── jwt/
├── web/                        # Embedded templates, static files, branding
├── docs/
├── migrations/                 # Numbered SQL files, embedded at build
├── scripts/                    # migrate, backup, test helpers
├── config.go                   # Config + DefaultConfig
├── validate.go
├── cache.go                    # CacheStore
├── cache_redis.go
├── cache_memory.go
├── migrate.go                  # RunMigrations / RunCoreMigrations
├── emailsender.go
└── sqlc.yaml
```

## Entry points

| File | Purpose |
|------|---------|
| `app/app.go` | What consumers import |
| `config.go` | `Config` types |
| `validate.go` | `ValidateConfig` |
| `cmd/api/main.go` | Env → Config → `app.New` |
| `examples/basic/main.go` | Smallest consumer |
| `internal/coreapp/app.go` | Repositories, services, handlers |

SQL for new queries goes in `internal/queries/`. After changing it, run `sqlc generate` and keep `internal/schema.sql` in sync with the live schema.

See [Architecture](ARCHITECTURE.md) for how the layers fit together.
