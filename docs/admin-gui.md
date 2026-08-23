# Admin GUI

Embedded HTMX panel. Path is `Config.Admin.AdminBasePath` (default `/gui`). No separate frontend build.

Branding (logo, colors, org name) is [web/README.md](../web/README.md).

## First account

```bash
go run ./cmd/setup
```

Username, email, password (masked). Stored with bcrypt. This is an admin account, not a tenant user.

## Login

`http://localhost:8080/gui/login` (or your `AdminBasePath`).

- Username and password
- Passkey, if one is registered
- Magic link, if enabled on that admin account

TOTP or email 2FA runs after the first factor when enabled.

JSON `/admin` routes are a different auth path: header `X-Admin-API-Key`.

## Pages

| Page | What it does |
|------|----------------|
| Dashboard | Counts and recent activity |
| Tenants / Applications | CRUD |
| OAuth Configs | Per-app Google / Facebook / GitHub |
| Users | Search, lock, sessions, social, trusted devices, CSV |
| Roles / Permissions / User Roles | End-user RBAC (`end_user_rbac`) |
| Operator IAM | `/gui/operator`. Roster, create and disable operators, stamp key roles, IAM events, access logs, custom roles |
| Sessions | Revoke one or many |
| Session Groups | Cross-app SSO, GlobalLogout |
| Activity Logs | Filter and export |
| API Keys | Admin and per-app keys, usage |
| Email Servers / Templates / Types | SMTP and mail content |
| Webhooks | Endpoints and deliveries |
| OIDC Clients | Relying parties, secret rotation |
| IP Rules | CIDR and country allow/block |
| Monitoring | Health + Prometheus summary |
| Settings | Overrides for CORS, logging, token TTLs |
| My Account | Password, 2FA, passkeys, magic link, trusted devices |

Default RBAC roles seeded on first migrate: `admin` (system) and `member` (system, assigned to new users).

## Session groups

Groups share login state across apps in a tenant. Each app is in at most one group. With `GlobalLogout` on, logout or session TTL in one member revokes the others. Real-time expiry needs Redis `--notify-keyspace-events Ex` (`Config.Session`). See [session group expiry](session-group-expiry.md).

## Stack

Go `html/template`, HTMX, Bootstrap 5, Bootstrap Icons. Assets are `go:embed`. No CDN.
