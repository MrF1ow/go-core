## Overview

The admin GUI is an HTMX-powered web interface mounted at the path configured by `Config.Admin.AdminBasePath` (default `/gui`). It provides a complete management dashboard for the Auth API platform. The main handler is `internal/admin/gui_handler.go` (the largest file in the project).

All paths below use the default `/gui` prefix. Replace with your configured `AdminBasePath` value if changed.

## Architecture

```
GUIHandler (internal/admin/gui_handler.go)
  |-- AccountService   -> admin auth, sessions, 2FA, CSRF
  |-- DashboardService -> stats aggregation
  |-- Repository       -> data access (tenants, apps, users, etc.)
  |-- SettingsService  -> system settings management
  |-- EmailService     -> email template/server management
  |-- RBACService      -> role/permission management
  |-- PasskeyService   -> WebAuthn for admin accounts
```

## Template Rendering (`web/renderer.go`)

**Embedded filesystem:** `//go:embed templates` in `web/renderer.go`

**Template structure:**
```
web/templates/
  layouts/     -> base layout files (*.tmpl)
  partials/    -> reusable fragments (*.tmpl) -- also served as standalone for HTMX
  pages/       -> full page templates (*.tmpl)
```

**Parsing strategy:**
- Full pages = layouts + partials + page file (rendered via `c.HTML()`)
- HTMX fragments = partials rendered standalone (for dynamic swaps)

**Template data struct (`web.TemplateData`):**
```go
type TemplateData struct {
    ActivePage       string      // Current page identifier for nav highlighting
    AdminUsername    string      // From auth context
    AdminID          string      // From auth context
    CSRFToken        string      // From CSRF middleware
    FlashSuccess     string      // Flash messages
    FlashError       string
    Error            string      // Login-specific error
    Username         string      // Pre-filled username on login error
    Redirect         string      // Post-login redirect URL
    TempToken        string      // 2FA temp token
    TwoFAMethod      string      // "totp" or "email"
    Data             interface{} // Page-specific arbitrary data
    OrgName          string      // Branding: org name (default "Auth API")
    LogoURL          string      // Branding: resolved logo URL
    PrimaryColor     string      // Branding: hex color for --bs-primary
    SecondaryColor   string      // Branding: hex color for --bs-secondary
    BorderRadius     string      // Branding: CSS length for --bs-border-radius
    SidebarColor     string      // Branding: hex sidebar background
    SidebarTextColor string      // Branding: hex sidebar text (auto-derived or explicit)
    CustomCSS        template.CSS // Branding: raw extra stylesheet (empty = off)
    FaviconURL       string      // Branding: resolved favicon URL (empty = shield data URI)
}
```

Branding fields are auto-populated by `Renderer.applyBranding()` on every `Instance()` call. `TemplateData` is overwritten. Empty keys on `gin.H` and `map[string]any` are filled. Handlers never set branding fields manually.

**Template functions available (`Renderer.defaultFuncMap`):**
- `basePath` -- returns the configured admin GUI path prefix (e.g. "/gui"). Used in templates as `{{basePath}}/login`
- `formatDate`, `formatDateTime`, `formatDateTimeFull` -- date formatting
- `timeAgo` -- human-readable relative time
- `upper`, `lower`, `title` -- string transforms
- `safeHTML`, `safeURL` -- mark content as safe (no escaping)
- `eq` -- string equality
- `deref` -- dereference *time.Time (nil-safe)
- `isExpired` -- check if time is in the past
- `add`, `sub` -- arithmetic for pagination

## Authentication Flow

### Standard Login
```
GET  /gui/login          -> LoginPage (render login form)
POST /gui/login          -> LoginSubmit [rate limited: 5/min]
  -> accountService.Authenticate(username, password)
  -> If 2FA: CreatePending2FASession -> redirect to /gui/2fa-verify
  -> If no 2FA: CreateSession -> set cookie -> redirect to /gui/
```

### 2FA Verification
```
GET  /gui/2fa-verify     -> TwoFAVerifyPage (render 2FA form)
POST /gui/2fa-verify     -> TwoFAVerifySubmit
  -> Validate TOTP code, email code, or recovery code
  -> PromotePendingSession -> set cookie -> redirect
```

### Passkey Login (passwordless)
```
POST /gui/passkey-login/begin  -> PasskeyLoginBegin (returns JSON challenge)
POST /gui/passkey-login/finish -> PasskeyLoginFinish (validates assertion)
  -> CreateSession -> set cookie -> return JSON success
```

### Magic Link Login (passwordless)
```
POST /gui/magic-link-login         -> MagicLinkLoginRequest
  -> Generate token in Redis (10 min) -> send email
GET  /gui/magic-link-login/verify  -> MagicLinkLoginVerify
  -> Validate token -> CreateSession -> set cookie -> redirect
```

### Session Cookie
- Name: `admin_session` (constant: `web.AdminSessionCookie`)
- Path: configured `AdminBasePath` (default `/gui`)
- Flags: HttpOnly, SameSite=Strict, Secure (auto-detected)
- Set via `web.SetSessionCookie(c, id, maxAge, basePath)`, cleared via `web.ClearSessionCookie(c, basePath)`

## HTMX CRUD Pattern

Every entity domain follows a consistent pattern. Example for tenants:

```
GET  /gui/tenants              -> TenantPage (full page with layout)
GET  /gui/tenants/list         -> TenantList (HTMX partial, paginated table)
GET  /gui/tenants/new          -> TenantCreateForm (HTMX partial, form)
POST /gui/tenants              -> TenantCreate (creates, returns alert + triggers list refresh)
GET  /gui/tenants/form-cancel  -> TenantFormCancel (empty response, cancels form)
GET  /gui/tenants/:id/edit     -> TenantEditForm (HTMX partial, pre-filled form)
PUT  /gui/tenants/:id          -> TenantUpdate (updates, returns alert + triggers list refresh)
GET  /gui/tenants/:id/delete   -> TenantDeleteConfirm (HTMX partial, confirmation modal)
DELETE /gui/tenants/:id        -> TenantDelete (deletes, triggers list refresh)
```

**HTMX signals:** Methods set `HX-Trigger` headers to signal events:
- Entity events: `tenantDeleted`, `roleDeleted`, `sessionListRefresh`, `socialAccountUnlinked`, `permissionsSaved`, etc.
- These trigger list refreshes and modal closes on the client side

**Error handling:** Errors are returned as Bootstrap alert HTML snippets via `c.String()` or `c.HTML()`.

## GUIHandler Method Groups

| Approx Lines | Section | Entity/Feature |
|------------|---------|----------------|
| 1-120 | Auth | Login, Logout, 2FA verify |
| 120-200 | Dashboard | Stats display |
| 200-550 | Tenants | Full CRUD |
| 550-950 | Applications | Full CRUD |
| 950-1350 | OAuth Configs | Full CRUD + toggle enabled |
| 1350-1900 | Users | List (paginated, searchable), detail, toggle active |
| 1900-2200 | Activity Logs | List (paginated, filterable), detail |
| 2200-2600 | API Keys | CRUD + revoke |
| 2600-2850 | Settings | Three-tier resolution display, update, reset |
| 2850-3050 | Email (Servers/Templates/Types) | Full CRUD for each |
| 3050-3100 | My Account (Email/Password) | Profile management |
| 3100-3330 | My Account (2FA) | TOTP generate/verify, email 2FA, recovery codes |
| 3335-3480 | My Account (Passkeys) | Register, delete, rename |
| 3480-3715 | Roles | CRUD + permission assignment modal |
| 3715-3865 | Permissions | List, create |
| 3865-4115 | User Roles | Assign, revoke, search users, dynamic dropdowns |
| 4115-4210 | Social Account Mgmt | Unlink (with lockout prevention) |
| 4215-4290 | Passkey Mgmt (for users) | Delete user's passkey |
| 4290-4340 | Helpers | `parseVariablesFromForm`, `escapeHTML` |
| 4340-4420 | Passkey Login | WebAuthn discoverable login ceremony |
| 4420-4600 | Magic Link Login | Request, verify |
| 4600-4976 | Session Management | List (cross-app), detail, revoke, per-user sessions |

## CSRF Protection

Middleware: `internal/middleware/csrf.go`

- GET requests: generate CSRF token, set in context as `csrf_token`
- POST/PUT/DELETE: read from `X-CSRF-Token` header (HTMX) or `_csrf` form field
- Token validated against session via `sessionValidator.ValidateCSRFToken()`

HTMX sends CSRF token via header: templates set `hx-headers='{"X-CSRF-Token": "{{.CSRFToken}}"}'`

## Rate Limiting (GUI-specific)

| Endpoint | Limit | Window | Lockout |
|----------|-------|--------|---------|
| POST /gui/login | 5/min | 60s | 10 -> 15min |
| POST /gui/passkey-login/* | 10/min | 60s | 20 -> 15min |
| POST /gui/magic-link-login | 3/15min | 15min | none |

GUI rate limiting sets `web.RateLimitErrorKey` in context (doesn't abort), letting the handler render the error in the form.

## Static Assets

Embedded via `web/static/embed.go` using `//go:embed`. Served at `<basePath>/static/*` (default `/gui/static/*`).

## Key Dependencies

- `admin.AccountService` implements `web.SessionValidator` (for GUI auth + CSRF middleware)
- `admin.Repository` implements `web.ApiKeyValidator` (for admin/app API key middleware)
- Both interfaces defined in `web/context_keys.go` to avoid import cycles

## Branding / Theming

Consumers customize admin dashboard appearance via `AdminBrandingConfig` in `core.Config`. Config-only, no DB migration. See [`web/README.md`](../../../web/README.md) for full docs.

**How it works:**
1. Consumer sets `cfg.Admin.Branding` fields (org name, logo, colors, border radius)
2. `ResolveBranding()` in `web/branding.go` resolves logo and favicon URLs and auto-derives sidebar text color
3. `Renderer.SetBranding()` stores resolved branding once at init
4. `Renderer.applyBranding()` injects branding fields into every `TemplateData` via `Instance()`. On `gin.H` and `map[string]any` it fills `FaviconURL`, `CustomCSS`, `PrimaryColor`, and `BorderRadius` only when those keys are missing or empty.
5. Templates use conditional `{{if .PrimaryColor}}` blocks to inject CSS variable overrides

**CSS injection pattern** (in `base.tmpl` and `login.tmpl`):
- Overrides `--bs-primary`, `--bs-secondary`, `--bs-border-radius` via inline `<style>` blocks
- `.btn-primary` hover/active states derived via `color-mix(in srgb, color 85%, black)`
- Sidebar colors applied via `.sidebar` class overrides with `!important`

**Logo resolution** (`LogoURL`):
- URL (`http://` or `https://`): used as-is in `<img src>`
- File path: file read once at startup, served from memory at `/gui/branding/logo`
- Content-Type detected via `http.DetectContentType`, with `.svg` → `image/svg+xml` and `.ico` → `image/x-icon`

**Favicon resolution** (`FaviconURL`):
- Empty: templates keep the built-in shield SVG data URI (does not fall back to `LogoURL`)
- URL (`http://` or `https://`): used as-is in `<link rel="icon" href>`
- File path: file read once at startup, served from memory at `/gui/branding/favicon`
- Same content-type rules as `LogoURL`

GUI CSP `img-src` is `'self' data: https:`, so remote `https://` branding URLs load. `http://` prefixes remain valid in config but are blocked by CSP.

**Sidebar text auto-derivation** (`web/branding.go`):
- When `SidebarColor` set but `SidebarTextColor` empty: compute WCAG relative luminance
- Luminance > 0.5 → dark text `#212529`, else → light text `#ffffff`

**Key files:**
- `web/branding.go` — `ParseHexColor`, `RelativeLuminance`, `AutoSidebarTextColor`, `ResolveBranding`
- `web/branding_test.go` — unit tests for luminance, hex parsing, auto-derive
- `web/templates/layouts/base.tmpl` — CSS injection block + sidebar logo/name
- `web/templates/pages/login.tmpl` — CSS injection + logo/name on login page
- `web/templates/pages/2fa_verify.tmpl` — CSS injection + logo on 2FA page
- `web/templates/pages/oidc_login.tmpl` — OIDC login; favicon, PrimaryColor, BorderRadius, CustomCSS fallback
- `web/templates/pages/oidc_consent.tmpl` — OIDC consent; same branding keys
- `web/templates/pages/oidc_logout.tmpl` — OIDC logout; same branding keys
- `web/templates/pages/oidc_error.tmpl` — OIDC error; same branding keys
- `web/templates/partials/favicon.tmpl` — shared favicon `<link>` (custom URL or shield data URI)

**Route:** `GET /gui/branding/logo` — registered only when `LogoURL` is a file path.
**Route:** `GET /gui/branding/favicon` — registered only when `FaviconURL` is a file path.

## When To Use This Skill

Load this skill when working on the admin web interface, HTMX templates, GUI authentication, branding/theming, or any `/gui/*` route.
