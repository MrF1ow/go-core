---
name: go-core-security
description: Security patterns and review checklist for the go-core auth API. Use this skill when making security-sensitive changes in go-core — touching authentication, authorization, JWT tokens, password handling, 2FA, session management, middleware, cookies, CORS, or any code that handles credentials or user identity. Also use when reviewing code for security issues, before running `make security`, or when the user mentions "security review", "hardening", or "vulnerability".
---

# Security Patterns for go-core Auth API

This is an authentication and authorization system — security isn't a feature, it's the product. Every change that touches auth flows, tokens, sessions, passwords, or user identity needs to follow these patterns.

## Authentication Architecture

### Token Lifecycle
- **Access tokens**: 15-minute TTL (configurable via `ACCESS_TOKEN_EXPIRATION_MINUTES`)
- **Refresh tokens**: 720-hour TTL (configurable via `REFRESH_TOKEN_EXPIRATION_HOURS`)
- **Token rotation**: New refresh token issued on each refresh
- **Blacklisting**: Redis-based, checked in `middleware.AuthMiddleware()`
- **Per-app TTL overrides**: Applications can override default token TTLs via DB settings

Token type is enforced — access tokens can't be used as refresh tokens and vice versa. See `pkg/jwt/jwt.go` for the claims structure.

### 2FA Flows
- **TOTP**: Time-based codes via authenticator apps, secrets stored encrypted
- **SMS**: Codes sent via configurable SMS provider (`internal/sms/`)
- **Email**: Codes sent via email service
- **Passkey/WebAuthn**: FIDO2 as a 2FA method or passwordless primary auth
- **Recovery codes**: One-time backup codes, bcrypt-hashed in DB
- **Trusted devices**: Cookie-based device trust with configurable SameSite policy

During 2FA verification, the user gets a temporary token (not a full access token) that can only be used at the 2FA verify endpoint.

### Social Auth (OAuth2)
- State parameter verified on callback to prevent CSRF
- Nonce validated for OIDC providers
- Account linking requires matching email + app context

## Patterns to Follow

### Password Handling
- Always bcrypt with `bcrypt.DefaultCost` (currently 10, but check `golang.org/x/crypto/bcrypt`)
- Never log passwords, even hashed ones
- Password reset tokens are single-use and time-limited
- Validate complexity in DTOs via validator tags, not in service layer

### Session Management
- Sessions tracked in Redis with user ID + app ID + device metadata
- `session.NewService()` handles creation, listing, and revocation
- Session groups enable cross-app SSO — logout in one app can revoke sessions across the group
- The `GroupLogoutFunc` callback in `internal/user/service.go` triggers group-wide revocation

### Middleware Stack
- `middleware.AuthMiddleware()` — JWT validation, user context extraction
- `middleware.CORSMiddleware()` — configurable CORS with admin GUI overrides
- `middleware.RateLimitMiddleware()` — per-endpoint rate limiting
- `middleware.IPRuleMiddleware()` — GeoIP and CIDR-based access control
- `middleware.CSRFMiddleware()` — timing-safe CSRF for admin GUI forms
- `middleware.APIKeyMiddleware()` — per-app API key validation with scopes

### Cookie Security
- Trusted device cookies use configurable SameSite via `TRUSTED_DEVICE_COOKIE_SAMESITE`
- The setting resolves through a 3-tier chain: env var > DB (admin GUI) > default ("none")
- `SameSite=None` requires `Secure=true` (enforced automatically)
- Session cookies follow the same pattern

### Error Responses
Never leak internal state through errors:
```go
// BAD: exposes DB schema
c.JSON(500, gin.H{"error": err.Error()})

// GOOD: generic message, log internally
log.Printf("widget creation failed: %v", err)
c.JSON(500, dto.ErrorResponse{Error: "An error occurred"})
```

Use `pkg/errors/NewAppError()` for structured error codes that map to HTTP status.

### Brute-Force Protection
- Account lockout after configurable failed attempts (per-app setting)
- Progressive delays between login attempts
- CAPTCHA trigger thresholds
- All managed by `internal/bruteforce/` service, wired into auth handlers

## Security Review Checklist

Before merging any auth-related change:

- [ ] No raw DB errors exposed to clients
- [ ] Passwords hashed with bcrypt (never plaintext, never reversible)
- [ ] JWT tokens validated for type (access vs refresh vs temporary)
- [ ] New endpoints added to correct route group (public vs protected vs admin)
- [ ] Multi-tenancy respected — queries filter by `AppID`
- [ ] Input validated via DTO validator tags before hitting service layer
- [ ] Sensitive fields use `json:"-"` in model structs
- [ ] Rate limiting applied to new auth endpoints
- [ ] Activity logging added for security-relevant events
- [ ] `make security` passes (gosec + nancy)
- [ ] No secrets in code or logs (check for hardcoded keys, tokens, passwords)
- [ ] CORS not widened beyond what's needed
- [ ] Timing-safe comparison used for token/code verification

## Running Security Scans

```bash
make security          # gosec + nancy (run before every commit)
make security-scan     # gosec only
make vulnerability-scan # nancy dependency audit only
```

Configuration in `.gosec.json` — medium severity threshold, includes test files.
