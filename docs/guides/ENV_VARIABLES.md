# Environment variables

The go-core **module** does not read the environment. These keys are used by:

1. `cmd/api`, which copies them onto `core.Config`
2. Activity logging and a few admin settings, which still call `os.Getenv` as one tier of their config chain (env, then database, then default)

Copy `.env.example` to `.env` when you run the reference app.

## `cmd/api` → `core.Config`

### Database

| Variable | Config field | Default |
|----------|--------------|---------|
| `DB_HOST` | `Database.Host` | `localhost` |
| `DB_PORT` | `Database.Port` | `5432` (`5433` in `.env.example` for Docker) |
| `DB_USER` | `Database.User` | `postgres` |
| `DB_PASSWORD` | `Database.Password` | empty |
| `DB_NAME` | `Database.DBName` | `go_core` |
| `DB_SSLMODE` | `Database.SSLMode` | `disable` |

### Redis

| Variable | Config field | Default |
|----------|--------------|---------|
| `REDIS_ADDR` | `Redis.Addr` | `localhost:6379` |
| `REDIS_PASSWORD` | `Redis.Password` | empty |
| `REDIS_DB` | `Redis.DB` | `1` |

There is no `REDIS_HOST` / `REDIS_PORT` split. Use `host:port` in `REDIS_ADDR`.

### JWT

| Variable | Config field | Default |
|----------|--------------|---------|
| `JWT_SECRET` | `JWT.Secret` | none (required, 32+ chars) |
| `ACCESS_TOKEN_EXPIRATION_MINUTES` | `JWT.AccessTokenTTL` | `15` |
| `REFRESH_TOKEN_EXPIRATION_HOURS` | `JWT.RefreshTokenTTL` | `720` |

There is no separate refresh-token secret. Both token types are signed with `JWT_SECRET`.

### CORS

| Variable | Default |
|----------|---------|
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:5173,http://localhost:5174,http://localhost:5175,http://localhost:8080` |
| `CORS_ALLOWED_METHODS` | `GET,POST,PUT,DELETE,OPTIONS,HEAD` |
| `CORS_ALLOWED_HEADERS` | includes `Authorization` and `X-App-ID` |
| `CORS_EXPOSE_HEADERS` | `Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Content-Type` |
| `CORS_MAX_AGE_HOURS` | `12` |
| `CORS_ALLOW_CREDENTIALS` | `true` |

### OIDC

| Variable | Default |
|----------|---------|
| `OIDC_ENABLED` | `false` |
| `OIDC_DEFAULT_APP_ID` | `00000000-0000-0000-0000-000000000001` |
| `OIDC_ID_TOKEN_EXPIRATION_MINUTES` | `60` |
| `OIDC_AUTH_CODE_EXPIRATION_MINUTES` | `10` |
| `PUBLIC_URL` | `http://localhost:8080` |

### WebAuthn

| Variable | Notes |
|----------|-------|
| `WEBAUTHN_RP_ID` | Relying party ID, usually the domain |
| `WEBAUTHN_RP_NAME` | Shown in browser prompts |
| `WEBAUTHN_RP_ORIGINS` | Comma-separated origins |

### SMS

| Variable | Notes |
|----------|-------|
| `SMS_PROVIDER` | Set to `twilio` to enable |
| `SMS_TWILIO_ACCOUNT_SID` | |
| `SMS_TWILIO_AUTH_TOKEN` | |
| `SMS_TWILIO_FROM_NUMBER` | E.164, e.g. `+15551234567` |

### Admin GUI

| Variable | Default |
|----------|---------|
| `ADMIN_API_KEY` | empty (JSON admin routes need this or a DB-backed key) |
| `ADMIN_EMAIL` | |
| `ADMIN_SESSION_EXPIRATION_HOURS` | `8` |
| `ADMIN_BASE_URL` | |
| `ADMIN_BASE_PATH` | `/gui` |
| `ADMIN_BRANDING_ORG_NAME` | |
| `ADMIN_BRANDING_LOGO_URL` | URL or local file path |
| `ADMIN_BRANDING_PRIMARY_COLOR` | hex |
| `ADMIN_BRANDING_SECONDARY_COLOR` | hex |
| `ADMIN_BRANDING_BORDER_RADIUS` | CSS length |
| `ADMIN_BRANDING_SIDEBAR_COLOR` | hex |
| `ADMIN_BRANDING_SIDEBAR_TEXT_COLOR` | hex, or auto from sidebar luminance |
| `ADMIN_BRANDING_CUSTOM_CSS` | raw CSS, max 64KiB. Multiline `.env` values are awkward; library consumers should set `Admin.Branding.CustomCSS` |
| `ADMIN_BRANDING_FAVICON_URL` | URL or local file path |

See [web/README.md](../../web/README.md) for branding rules.

### Social redirects

| Variable | Config field |
|----------|--------------|
| `ALLOWED_REDIRECT_DOMAINS` | `Social.AllowedRedirectDomains` |
| `DEFAULT_REDIRECT_URI` | `Social.DefaultRedirectURI` |

OAuth **client IDs and secrets** are not env vars. Configure them per app in the admin GUI.

### Other

| Variable | Default |
|----------|---------|
| `GEOIP_DB_PATH` | empty |
| `TRUSTED_DEVICE_COOKIE_SAMESITE` | `none` |
| `SESSION_GROUP_EXPIRY_REVOCATION_ENABLED` | `false` |
| `SESSION_GROUP_EXPIRY_SCAN_INTERVAL` | `5m` |
| `SESSION_GROUP_KEYSPACE_NOTIF_ENABLED` | `false` |
| `REDIS_NOTIFY_KEYSPACE_EVENTS` | empty (`Ex` for real-time expiry) |
| `MULTI_TENANT` | `false` |
| `FRONTEND_URL` | `http://localhost:5173` |
| `APP_NAME` | empty |
| `PORT` | `8080` |
| `GIN_MODE` | `debug` |

## Activity logging (read directly)

These are not `core.Config` fields. `internal/config/logging.go` reads them at process start. The admin settings page can override the same keys in the database.

| Variable | Default |
|----------|---------|
| `LOG_TOKEN_REFRESH` | `false` |
| `LOG_PROFILE_ACCESS` | `false` |
| `LOG_SAMPLE_TOKEN_REFRESH` | `0.01` |
| `LOG_SAMPLE_PROFILE_ACCESS` | `0.01` |
| `LOG_DISABLED_EVENTS` | empty (comma-separated event names) |
| `LOG_ANOMALY_DETECTION_ENABLED` | `true` |
| `LOG_ANOMALY_NEW_IP` | `true` |
| `LOG_ANOMALY_NEW_USER_AGENT` | `true` |
| `LOG_ANOMALY_GEO_CHANGE` | `false` |
| `LOG_ANOMALY_UNUSUAL_TIME` | `false` |
| `LOG_ANOMALY_SESSION_WINDOW` | `720h` |
| `LOG_RETENTION_CRITICAL` | `365` (days) |
| `LOG_RETENTION_IMPORTANT` | `180` |
| `LOG_RETENTION_INFORMATIONAL` | `90` |
| `LOG_CLEANUP_ENABLED` | `true` |
| `LOG_CLEANUP_INTERVAL` | `24h` |
| `LOG_CLEANUP_BATCH_SIZE` | `1000` |
| `LOG_ARCHIVE_BEFORE_CLEANUP` | `false` |

Durations use Go syntax (`24h`, `15m`). Booleans accept `true`/`false`, `1`/`0`, `yes`/`no`.

## Email

`cmd/api` does not copy SMTP variables into `Config.Email`. Set `cfg.Email` in code, or add an SMTP server in the admin GUI (Email Servers). Integration tests look for `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` when you run them against a real mailer.

## See also

- [Configuration](../configuration.md)
- [Activity logging](../activity-logging.md)
