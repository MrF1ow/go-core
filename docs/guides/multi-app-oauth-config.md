# OAuth redirect domains

Social login (Google, Facebook, GitHub) is the same set of routes for every application. After the provider callback, the API redirects the browser to a frontend URI. That URI must be on an allowed domain.

Provider client IDs and secrets are **not** set here. They are stored per application in the database (Admin GUI → OAuth Configs, or `POST /admin/oauth-providers`).

## Config

```go
cfg.Social = core.SocialConfig{
    AllowedRedirectDomains: []string{
        "localhost:5173",
        "app.example.com",
        ".example.com",
    },
    DefaultRedirectURI: "https://app.example.com/auth/callback",
}
```

In the reference app:

```bash
ALLOWED_REDIRECT_DOMAINS=localhost:5173,app.example.com,.example.com
DEFAULT_REDIRECT_URI=https://app.example.com/auth/callback
```

A leading `.` allows all subdomains of that host.

## Login URLs

```
GET /auth/google/login
GET /auth/facebook/login
GET /auth/github/login
```

Append `?redirect_uri=` to send the tokens somewhere other than `DefaultRedirectURI`. The host of that URI must match the allowlist.

```javascript
window.location.href =
  "/auth/google/login?redirect_uri=" +
  encodeURIComponent("https://admin.example.com/auth/callback");
```

The callback page reads `access_token`, `refresh_token`, `provider`, or `error` from the query string.

## Errors on the redirect

| Query | Meaning |
|-------|---------|
| `error=invalid_state` | OAuth state missing, expired, or tampered |
| `error=authorization_code_missing` | Provider did not return a code |
| `error=missing_state` | State parameter absent |
| `error=Invalid%20redirect%20URI` | Host not on the allowlist |

The state parameter is HMAC-signed and includes a timestamp so it cannot be replayed as an open redirect.
