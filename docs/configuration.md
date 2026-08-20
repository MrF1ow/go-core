# Configuration

Consumers build a `core.Config` and pass it to `app.New()`. The module does not read environment variables. The reference app in `cmd/api` is the exception: it maps a `.env` file onto this struct. Those keys are listed in [Environment variables](guides/ENV_VARIABLES.md).

```go
cfg := core.DefaultConfig()
cfg.Database.Host = "localhost"
cfg.Database.DBName = "myapp"
cfg.Database.User = "postgres"
cfg.Database.Password = "secret"
cfg.JWT.Secret = "your-secret-at-least-32-characters-long"

app, err := app.New(cfg)
```

`DefaultConfig()` fills CORS allowlists and token lifetimes. Required fields that are still empty cause `app.New()` to return an error.

Some runtime values (activity log retention, CORS, OAuth redirect domains) can later be overridden in the admin GUI. Resolution is config value, then database setting, then default.

## Required

| Field | Notes |
|-------|-------|
| `Database.Host` | PostgreSQL host |
| `Database.Port` | Default 5432 |
| `Database.DBName` | Database name |
| `Database.User` | Database user |
| `Database.Password` | Not validated (empty is allowed for local trust auth) |
| `JWT.Secret` | Signing key, minimum 32 characters |

`JWT.AccessTokenTTL` defaults to 15 minutes. `JWT.RefreshTokenTTL` defaults to 720 hours.

## Optional

| Field | When unset |
|-------|------------|
| `Redis` | `nil` uses an in-memory store. Fine for tests. Use Redis in production for sessions and token blacklists. |
| `Email` | `nil` disables sending. Magic links, verification, and email 2FA will not work until you set SMTP or configure a server in the admin GUI. The reference app does not map SMTP env vars into this field. |
| `CORS` | Localhost origins and common headers from `DefaultConfig()` |
| `OIDC` | Disabled. Set `OIDC.Enabled` and `PublicURL` to issue ID tokens. |
| `WebAuthn` | Empty `RPID` disables passkeys. Set `RPID`, `RPName`, and `RPOrigins`. |
| `SMS` | Empty `Provider` disables SMS 2FA. Twilio needs account SID, auth token, and from-number. |
| `Admin` | Empty `APIKey` still allows GUI login after `make setup-admin`. The key is for `/admin` JSON routes (`X-Admin-API-Key`). `AdminBasePath` defaults to `/gui`. Branding is documented in [web/README.md](../web/README.md). |
| `Social` | `AllowedRedirectDomains` and `DefaultRedirectURI` for OAuth callbacks. Provider client IDs live per-app in the database, not on this struct. |
| `GeoIP` | Empty `DBPath` skips country lookups. CIDR rules still work. |
| `Session` | Session-group expiry is off by default. See [session group expiry](session-group-expiry.md). |
| `MultiTenant` | `false`. Missing `X-App-ID` uses the default app. `true` requires the header. |
| `PublicURL` | Used in OIDC discovery and some email links. |
| `FrontendURL` | Redirect target after OAuth when no `redirect_uri` is given. |
| `AppName` | Shown in emails and the admin GUI when branding `OrgName` is empty. |

`Port` and `GinMode` are only used by `cmd/api`.

## Database

```go
cfg.Database = core.DatabaseConfig{
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "secret",
    DBName:   "myapp",
    SSLMode:  "disable",
}
```

## Redis

```go
cfg.Redis = &core.RedisConfig{
    Addr:     "localhost:6379",
    Password: "",
    DB:       1,
}
```

Set `cfg.Redis = nil` for the in-memory store.

## Email

```go
cfg.Email = &core.EmailConfig{
    Host:     "smtp.example.com",
    Port:     587,
    Username: "noreply@example.com",
    Password: "app-password",
    From:     "noreply@example.com",
    UseTLS:   true,
}
```

Per-app SMTP servers configured in the admin GUI take precedence for that app.

## WebAuthn

```go
cfg.WebAuthn = core.WebAuthnConfig{
    RPID:      "example.com",
    RPName:    "My App",
    RPOrigins: []string{"https://example.com", "https://app.example.com"},
}
```

Locally, `RPID` is usually `localhost` and origins include `http://localhost:8080`.

## OIDC

```go
cfg.PublicURL = "https://auth.example.com"
cfg.OIDC = core.OIDCConfig{
    Enabled:      true,
    DefaultAppID: "00000000-0000-0000-0000-000000000001",
    IDTokenTTL:   60 * time.Minute,
    AuthCodeTTL:  10 * time.Minute,
}
```

RSA keys for RS256 ID tokens are generated per application and stored in the database.

## SMS

```go
cfg.SMS = core.SMSConfig{
    Provider:         "twilio",
    TwilioAccountSID: "ACxxxxxxxx",
    TwilioAuthToken:  "token",
    TwilioFromNumber: "+15551234567",
}
```

## GeoIP

```go
cfg.GeoIP.DBPath = "/data/GeoLite2-City.mmdb"
```

Country-based IP rules need this file. CIDR rules do not.

## Social login

Provider credentials (Google, Facebook, GitHub) are stored per application via the admin GUI or `POST /admin/oauth-providers`. `Config.Social` only controls which redirect URIs are accepted after the callback. See [OAuth redirect domains](guides/multi-app-oauth-config.md).

## Activity logging

Log retention, sampling, and anomaly flags are not fields on `core.Config`. They are resolved from env vars and the admin settings UI. See [Activity logging](activity-logging.md).
