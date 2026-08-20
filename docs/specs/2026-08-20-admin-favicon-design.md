# Admin GUI favicon

**Date:** 2026-08-20
**Approach:** One `FaviconURL` field on `AdminBrandingConfig`, same URL-or-file-path rules as `LogoURL`. Empty keeps the existing shield data URI.

## Summary

Consumers can set the admin GUI tab icon at startup. The module does not store a favicon in the database. Custom CSS cannot do this: a tab icon is a `<link rel="icon">` in `<head>`, not a stylesheet rule.

`LogoURL` is not reused. Logos are often wide and look bad at 16×16. If a consumer wants the same file for both, they set both fields to the same path.

## Config

Add one field to the existing branding struct:

```go
type AdminBrandingConfig struct {
    OrgName          string
    LogoURL          string
    PrimaryColor     string
    SecondaryColor   string
    BorderRadius     string
    SidebarColor     string
    SidebarTextColor string
    CustomCSS        string
    FaviconURL       string // URL or file path. Empty = shield data URI
}
```

```go
cfg.Admin.Branding.FaviconURL = "https://acme.com/favicon.ico"
// or
cfg.Admin.Branding.FaviconURL = "/var/branding/favicon.svg"
```

## URL vs file path

Same split as `LogoURL`:

| Value | Behavior |
|-------|----------|
| Empty | Templates keep the current shield SVG data URI |
| `http://` or `https://` | Used as-is in `<link rel="icon" href="...">` |
| Anything else | Treated as a local file. Read once at `app.New()`. Served from memory at `<AdminBasePath>/branding/favicon` |

Do not parse `data:` as a third mode. A consumer who wants an inline SVG reads a file or hosts a URL.

## Injection

Pass `FaviconURL` through `ResolvedBranding` and `Renderer.applyBranding` the same way as `LogoURL`. After a file-path resolve, `TemplateData.FaviconURL` is the serve URL, not the disk path.

On the three admin shells, replace the hardcoded icon link:

```html
{{if .FaviconURL}}
    <link rel="icon" href="{{.FaviconURL}}">
{{else}}
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,...existing shield...">
{{end}}
```

Pages that get the tag (same shells as branding CSS / CustomCSS):

- `web/templates/layouts/base.tmpl`
- `web/templates/pages/login.tmpl`
- `web/templates/pages/2fa_verify.tmpl`

OIDC login, consent, logout, and error pages stay out of scope. Those already have per-client logos.

Do not set `type` on the custom `<link>`. Remote URLs do not tell us the MIME type; browsers sniff. The default data URI keeps `type="image/svg+xml"`.

Do not add `apple-touch-icon`, `sizes`, or a dark/light pair. One `rel="icon"` is enough.

`html/template` already escapes `href`. Do not mark the URL as `template.URL`.

## File serving

Mirror the logo route.

At `app.New()`, if `FaviconURL` is a file path, `os.ReadFile` and store bytes plus content type on `coreapp.App`. Register `GET <AdminBasePath>/branding/favicon` only in that case.

```
Cache-Control: public, max-age=86400, immutable
```

Content type:

1. Suffix `.svg` (any case) → `image/svg+xml` (`http.DetectContentType` gets SVG wrong)
2. Suffix `.ico` (any case) → `image/x-icon`
3. Otherwise `http.DetectContentType` on the bytes

Extract a small helper used by both logo and favicon so the SVG/ICO override and the `http://` / `https://` check are not copied. Suggested shape, local to `internal/coreapp` or `web`:

```go
func isRemoteAssetURL(s string) bool {
    return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func readBrandingFile(path string) (data []byte, contentType string, err error)
```

Do not share in-memory bytes when logo and favicon point at the same path. Two reads is fine.

## `ResolveBranding` arity

`ResolveBranding` already takes nine strings. Do not add a tenth and eleventh positional argument.

Change it to a struct. Package `web` is not the module's public API (`app.New` / `core.Config` are). Existing call sites are `internal/coreapp` and `web` tests.

```go
type BrandingInput struct {
    OrgName          string
    LogoURL          string
    PrimaryColor     string
    SecondaryColor   string
    BorderRadius     string
    SidebarColor     string
    SidebarTextColor string
    CustomCSS        string
    FaviconURL       string
    LogoServeURL     string
    FaviconServeURL  string
}

func ResolveBranding(in BrandingInput) ResolvedBranding
```

Rules stay the same: non-empty `LogoServeURL` / `FaviconServeURL` replace the corresponding URL; sidebar text auto-derives as today.

## Validation

`validateBranding` in `validate.go`, called from `ValidateConfig` / `app.New()`.

Copy the `LogoURL` file rules onto `FaviconURL`. Empty string skips the checks. Remote URLs are not fetched at startup.

| Rule | On failure |
|------|------------|
| File path and `os.Stat` fails | Error naming `FaviconURL` and the path |
| File larger than 1 MiB | Error naming `FaviconURL` and the limit |

Do not validate file extension. ICO, PNG, SVG, GIF, WebP, JPEG are all fine in a browser `<link rel="icon">`. A bad file is the consumer's problem; the tab shows the browser default.

Extract the 1 MiB cap so logo and favicon share one constant.

## CSP

GUI `img-src` is currently `'self' data:`. That already breaks documented `LogoURL` `https://` values (`<img src>` is subject to `img-src`). Favicon `<link rel="icon">` is inconsistently covered by CSP across browsers.

Widen GUI `img-src` to `'self' data: https:` — the same token OIDC pages already use for client logos. Branding URLs are set by the deployer at startup, not by end users.

Do not add `http:` to CSP. Remote branding URLs should be HTTPS. Leave `LogoURL` / `FaviconURL` validation allowing `http://` prefixes (no behavior change on that check); those HTTP images still will not load under CSP.

No other CSP change. `font-src` stays as it is.

## `cmd/api`

Map `ADMIN_BRANDING_FAVICON_URL` onto `Admin.Branding.FaviconURL`, same pattern as `ADMIN_BRANDING_LOGO_URL`. Document it in `docs/guides/ENV_VARIABLES.md` and comment it in `.env.example`.

## Tests

Add to `validate_test.go`:

- empty `FaviconURL` is valid
- `https://example.com/favicon.ico` is valid (no file check)
- missing file path is rejected with `FaviconURL` in the error
- file larger than 1 MiB is rejected with `FaviconURL` in the error

Add to `web/branding_test.go` (or renderer tests):

- `ResolveBranding` copies `FaviconURL`
- non-empty `FaviconServeURL` wins
- `applyBranding` sets `TemplateData.FaviconURL`
- login HTML with `FaviconURL` set emits that href and does not emit the shield data URI
- login HTML with empty `FaviconURL` still emits the shield data URI

Add to `internal/middleware/security_headers_test.go`:

- GUI CSP contains `img-src 'self' data: https:`

No screenshot or browser tests. Do not add a favicon route integration test unless one already exists for `/branding/logo` to copy.

## Docs

- `web/README.md`: field row, URL-vs-file rules, serve path, CSP `https:` note (also applies to `LogoURL`)
- `docs/guides/ENV_VARIABLES.md` and `.env.example`
- `examples/basic/main.go`: commented `FaviconURL` next to `LogoURL`
- `.claude/skills/go-core/references/admin-gui.md`: `FaviconURL` on `TemplateData`, serve route
- `.claude/skills/go-core/references/integration.md`: include `FaviconURL` in the branding snippet

## Out of scope

- Falling back to `LogoURL`
- `data:` URLs as a first-class input
- `apple-touch-icon` / mask icons / `sizes`
- Separate light and dark favicons
- OIDC page injection
- Database-backed branding or a GUI upload
- Fetching remote favicons at startup to re-host them
- Changing `font-src`

## Files

- `config.go`
- `validate.go`, `validate_test.go`
- `web/branding.go`, `web/branding_test.go`
- `web/renderer.go`
- `web/templates/layouts/base.tmpl`
- `web/templates/pages/login.tmpl`
- `web/templates/pages/2fa_verify.tmpl`
- `internal/coreapp/app.go`
- `internal/middleware/security_headers.go`, `security_headers_test.go`
- `cmd/api/main.go`
- `web/README.md`
- `docs/guides/ENV_VARIABLES.md`
- `.env.example`
- `examples/basic/main.go`
- `.claude/skills/go-core/references/admin-gui.md`
- `.claude/skills/go-core/references/integration.md`
