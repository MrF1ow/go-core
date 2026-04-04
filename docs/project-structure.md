# Project Structure

```
project-root/
├── app/                        # Public entry point (New, RegisterRoutes, Close)
│   └── app.go
├── cmd/
│   ├── api/                    # Reference application entry point
│   │   └── main.go
│   ├── migrate_oauth/          # OAuth migration tool
│   │   └── main.go
│   └── setup/                  # Admin account setup wizard
│       └── main.go
├── examples/
│   └── basic/                  # Minimal consumer example
│       └── main.go
├── internal/                   # Private application code
│   ├── coreapp/                # Internal wiring (not accessible to consumers)
│   ├── admin/                  # Admin API (tenant/app/OAuth management) + Admin GUI
│   ├── auth/                   # Authentication handlers
│   ├── user/                   # User management (includes magic link login, import/export)
│   ├── social/                 # Social OAuth2 providers + social account linking
│   ├── twofa/                  # Two-factor authentication (TOTP, email, SMS, backup email, trusted devices)
│   ├── webauthn/               # WebAuthn/passkey registration, 2FA, and passwordless login
│   ├── rbac/                   # Role-based access control (roles, permissions, user-roles)
│   ├── session/                # Session management (list/revoke active sessions)
│   ├── sessiongroup/           # Cross-app SSO session linking and global logout
│   ├── oidc/                   # OIDC provider (Authorization Code + PKCE, RS256 ID tokens, JWKS)
│   ├── webhook/                # Webhook system (endpoint registry, async delivery queue, retries)
│   ├── bruteforce/             # Brute-force protection (account lockout, progressive delays, CAPTCHA)
│   ├── geoip/                  # GeoIP service (MaxMind) + IP access rules (CIDR/country per app)
│   ├── health/                 # Health check + Prometheus metrics endpoint
│   ├── sms/                    # SMS sender interface + Twilio implementation
│   ├── log/                    # Activity logging system
│   ├── email/                  # Email verification & reset
│   ├── middleware/             # JWT auth, AppID, CORS, rate limiting, security headers, session validation
│   ├── database/               # Database connection (pgx pool)
│   ├── redis/                  # Redis connection & operations
│   ├── settings/               # Runtime settings service
│   └── util/                   # Utility functions
├── pkg/                        # Public packages
│   ├── models/                 # Database models
│   ├── dto/                    # Data transfer objects
│   ├── errors/                 # Custom error types
│   └── jwt/                    # JWT token utilities
├── web/                        # Embedded HTMX templates and static assets (go:embed)
├── docs/                       # Documentation
│   └── guides/                 # Setup and configuration guides
├── migrations/                 # SQL migration files
├── scripts/                    # Helper scripts (migrate, backup, cleanup)
├── .github/                    # GitHub configuration (CI/CD, issue templates)
├── config.go                   # Config struct and DefaultConfig()
├── validate.go                 # Config validation (ValidateConfig)
├── cache.go                    # CacheStore interface
├── cache_redis.go              # Redis CacheStore implementation
├── cache_memory.go             # In-memory CacheStore implementation
├── migrate.go                  # RunMigrations helper
├── emailsender.go              # EmailSender interface
├── Dockerfile                  # Production Docker image
├── Dockerfile.dev              # Development Docker image
├── docker-compose.yml          # Production compose config
├── docker-compose.dev.yml      # Development compose config
├── Makefile                    # Build and development commands
├── .air.toml                   # Hot reload configuration
├── .env.example                # Environment variables template
├── go.mod / go.sum             # Go module dependencies
└── LICENSE                     # MIT License
```

---

## Key Files

| File | Purpose |
|------|---------|
| `app/app.go` | Public entry point — `New()`, `RegisterRoutes()`, `Close()` |
| `config.go` | `Config` struct, `DefaultConfig()`, all configuration types |
| `validate.go` | `ValidateConfig()` — fails fast on missing required fields |
| `cmd/api/main.go` | Reference application — full setup with env var loading |
| `examples/basic/main.go` | Minimal consumer example |
| `internal/coreapp/app.go` | Internal wiring — initializes all services and repositories |
| `pkg/models/` | Database models (shared across all modules) |
| `pkg/dto/` | API request/response data transfer objects |
| `pkg/errors/errors.go` | Custom error types (`NewAppError` with HTTP status codes) |
| `pkg/jwt/jwt.go` | JWT token creation and validation |
| `.env.example` | Environment variable template |

---

## Architecture

Each domain follows the **Repository-Service-Handler** pattern:

- **Repository** — Data access via pgx (connection pool) and SQLC (generated queries)
- **Service** — Business logic, validation, orchestration
- **Handler** — HTTP transport, request binding, response formatting

Cross-domain dependencies use callback function fields (set during wiring in `internal/coreapp/`) to avoid circular imports.

For the full architecture documentation, see [ARCHITECTURE.md](ARCHITECTURE.md).
