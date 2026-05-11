# Admin Dashboard Branding

**Date:** 2026-05-11
**Approach:** CSS Variables Injection (mirrors existing OIDC page theming pattern)

## Summary

Allow consumers of the go-core module to customize the admin dashboard appearance via config struct fields. Branding covers: org name, logo, primary/secondary colors, border radius, sidebar color, and sidebar text color. No DB migration — config-only, set at startup.

## Config Struct

New `AdminBrandingConfig` nested inside `AdminConfig`:

```go
type AdminBrandingConfig struct {
    OrgName          string // Replaces "Auth API" text on login page + sidebar. Default: "Auth API"
    LogoURL          string // URL or file path to org logo. Login page (64px) + sidebar (32px). Empty = shield icon fallback
    PrimaryColor     string // Hex "#RRGGBB" or "#RGB". Overrides --bs-primary. Empty = Bootstrap default (#0d6efd)
    SecondaryColor   string // Hex color. Overrides --bs-secondary. Empty = Bootstrap default
    BorderRadius     string // CSS length ("0.5rem", "0", "1rem"). Overrides --bs-border-radius. Empty = Bootstrap default
    SidebarColor     string // Hex bg color for sidebar. Empty = default (#212529 light, #101214 dark)
    SidebarTextColor string // Hex text color for sidebar. Empty = auto-derived from SidebarColor luminance
}
```

Added to existing `AdminConfig`:

```go
type AdminConfig struct {
    APIKey     string
    Email      string
    SessionTTL time.Duration
    BaseURL    string
    Branding   AdminBrandingConfig
}
```

Consumer usage:

```go
cfg := core.Config{
    Admin: core.AdminConfig{
        Branding: core.AdminBrandingConfig{
            OrgName:      "Acme Corp",
            LogoURL:      "https://acme.com/logo.svg",  // or "./assets/logo.png"
            PrimaryColor: "#4f46e5",
            BorderRadius: "0.75rem",
            SidebarColor: "#1a1a2e",
        },
    },
}
app, err := app.New(cfg)
```

Zero-value `AdminBrandingConfig` = no branding overrides, default Bootstrap appearance.

## LogoURL Resolution

`LogoURL` accepts two forms:

1. **External URL** (`http://` or `https://` prefix) — used as-is in template `<img src>` attributes
2. **File path** (anything else) — file is served at `/gui/branding/logo` route; templates receive that internal URL

Resolution happens once in `ResolveBranding()` at startup:
- Check if `LogoURL` starts with `http://` or `https://`
- If yes: store as-is
- If no: validate file exists and is readable, store original path for serving, set resolved URL to `/gui/branding/logo`

Route registration: when `LogoURL` is a file path, `RegisterRoutes` adds a `GET /gui/branding/logo` handler that serves the file with appropriate `Content-Type` detection via `http.DetectContentType`. The file is read once at startup and served from memory.

Validation at `app.New()`: if `LogoURL` is a file path, the file must exist and be readable. Return error if not.

## TemplateData Pass-through

Add branding fields to `web.TemplateData`:

```go
type TemplateData struct {
    // ... existing fields ...

    OrgName          string
    LogoURL          string
    PrimaryColor     string
    SecondaryColor   string
    BorderRadius     string
    SidebarColor     string
    SidebarTextColor string // resolved: explicit config value OR auto-derived
}
```

`web.Renderer` gains a stored `branding` struct set once via `SetBranding()` during init. Every `Instance()` call auto-populates branding fields on the `TemplateData` — individual handlers never set branding manually.

## Sidebar Text Auto-Derivation

When `SidebarColor` is set but `SidebarTextColor` is empty, compute text color automatically:

1. Parse hex color to RGB
2. Compute WCAG relative luminance: `L = 0.2126*R' + 0.7152*G' + 0.0722*B'` (with sRGB linearization)
3. If `L > 0.5` → dark text `"#212529"`, else → light text `"#ffffff"`

Implementation lives in `web/branding.go` — pure Go, no dependencies.

When `SidebarColor` is empty (defaults used), `SidebarTextColor` config is ignored — default sidebar always uses white text.

## Template Changes

### `base.tmpl` — CSS Variable Injection

After the existing `<style>` block, add a conditional branding block:

```html
{{if or .PrimaryColor .SecondaryColor .BorderRadius .SidebarColor}}
<style>
    :root {
        {{if .PrimaryColor}}--bs-primary: {{.PrimaryColor}};{{end}}
        {{if .SecondaryColor}}--bs-secondary: {{.SecondaryColor}};{{end}}
        {{if .BorderRadius}}--bs-border-radius: {{.BorderRadius}};{{end}}
    }
    {{if .PrimaryColor}}
    .btn-primary {
        --bs-btn-bg: {{.PrimaryColor}};
        --bs-btn-border-color: {{.PrimaryColor}};
        --bs-btn-hover-bg: color-mix(in srgb, {{.PrimaryColor}} 85%, black);
        --bs-btn-hover-border-color: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
        --bs-btn-active-bg: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
    }
    {{end}}
    {{if .SidebarColor}}
    .sidebar { background-color: {{.SidebarColor}} !important; }
    .sidebar .nav-link { color: {{.SidebarTextColor}}; opacity: 0.75; }
    .sidebar .nav-link:hover { color: {{.SidebarTextColor}}; opacity: 1; }
    .sidebar .nav-link.active { color: {{.SidebarTextColor}}; opacity: 1; }
    .sidebar-heading { color: {{.SidebarTextColor}}; opacity: 0.5; }
    [data-bs-theme="dark"] .sidebar { background-color: {{.SidebarColor}} !important; }
    {{end}}
</style>
{{end}}
```

### `base.tmpl` — Sidebar Header

Replace hardcoded "Auth API" with:

```html
{{if .LogoURL}}<img src="{{.LogoURL}}" alt="{{.OrgName}}" style="height:32px;width:32px;object-fit:contain;" class="me-2">{{end}}
<span class="fs-5 fw-bold text-white">{{if .OrgName}}{{.OrgName}}{{else}}Auth API{{end}}</span>
```

### `login.tmpl` — Branding

1. CSS variable injection (same primary color + border radius block)
2. Replace shield icon:

```html
{{if .LogoURL}}
<img src="{{.LogoURL}}" alt="{{.OrgName}}" style="height:64px;width:64px;object-fit:contain;">
{{else}}
<i class="bi bi-shield-lock text-primary" style="font-size: 3rem;"></i>
{{end}}
```

3. Replace "Auth API" / "Admin Panel" text:

```html
<h3 class="mt-2 fw-bold">{{if .OrgName}}{{.OrgName}}{{else}}Auth API{{end}}</h3>
<p class="text-muted">Admin Panel</p>
```

### `2fa_verify.tmpl`

Same pattern as login.tmpl if it has a standalone layout (logo + org name + primary color).

## Validation

At `app.New()` time, validate branding config before proceeding:

| Field | Rule | On failure |
|-------|------|------------|
| `PrimaryColor` | Valid hex (`#RGB` or `#RRGGBB`) or empty | Return error |
| `SecondaryColor` | Valid hex or empty | Return error |
| `SidebarColor` | Valid hex or empty | Return error |
| `SidebarTextColor` | Valid hex or empty | Return error |
| `BorderRadius` | Valid CSS length (digits + rem/px/em) or empty | Return error |
| `LogoURL` | If URL: no validation. If file path: must exist and be readable | Return error |
| `OrgName` | No validation (any string) | — |

Invalid config = `app.New()` returns error. No silent fallback to defaults for bad values.

## New Files

- `web/branding.go` — `ParseHexColor`, `RelativeLuminance`, `AutoSidebarTextColor`, `ResolveBranding`, hex validation, logo file loading
- `web/branding_test.go` — unit tests for luminance calc, hex parsing, auto-derive logic, validation
- `web/README.md` — documents AdminBrandingConfig fields, usage examples, and theming behavior

## Modified Files

- `config.go` — add `AdminBrandingConfig` struct, add `Branding` field to `AdminConfig`
- `web/renderer.go` — add branding fields to `TemplateData`, add `SetBranding()`, populate in `Instance()`
- `web/templates/layouts/base.tmpl` — CSS injection block, sidebar header logo/name
- `web/templates/pages/login.tmpl` — CSS injection, logo/name replacement
- `web/templates/pages/2fa_verify.tmpl` — same as login if standalone layout
- `internal/coreapp/app.go` — validate branding config, call `renderer.SetBranding()` during init
- `app/app.go` — pass branding config through to coreapp
- `internal/coreapp/app.go` — register `/gui/branding/logo` route when LogoURL is a file path
- `README.md` — reference web/README.md for admin branding docs
- `CLAUDE.md` — reference web/README.md for admin branding docs

## Documentation

### `web/README.md`

New file documenting:
- All `AdminBrandingConfig` fields with types, defaults, and examples
- LogoURL resolution behavior (URL vs file path)
- Sidebar text color auto-derivation logic
- CSS variables that get overridden
- Complete consumer usage example

### Main `README.md` + `CLAUDE.md`

Add brief reference to `web/README.md` under admin GUI section, e.g.:
> See [`web/README.md`](web/README.md) for admin dashboard branding and theming configuration.

## Out of Scope

- DB-backed branding (runtime changes without redeploy)
- Configurable admin path prefix (separate effort, 311 hardcoded references)
- Favicon customization
- Custom CSS file upload
- Font customization
