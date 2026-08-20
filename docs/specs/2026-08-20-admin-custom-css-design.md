# Admin GUI custom CSS

**Date:** 2026-08-20
**Approach:** One raw CSS string on `AdminBrandingConfig`, injected after the existing branding `<style>` block. Light and dark live in that same stylesheet via `[data-bs-theme]`.

## Summary

Consumers can attach extra CSS to the admin GUI at startup. The module does not read a file and does not store CSS in the database. Fonts, favicon, and extra config fields can wait. If a consumer wants those later, this field is already enough for `@font-face` and theme-specific overrides.

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
    CustomCSS        string // raw CSS. Empty = off
}
```

Empty string means no extra `<style>` tag.

```go
cfg.Admin.Branding.CustomCSS = `
:root { --brand-font: "Source Sans 3", system-ui, sans-serif; }
body { font-family: var(--brand-font); }

[data-bs-theme="light"] {
    --bs-body-bg: #f7f5f2;
}

[data-bs-theme="dark"] {
    --bs-body-bg: #121212;
    --bs-primary: #818cf8;
}
`
```

The library does not load CSS from disk. A consumer who keeps CSS in a file reads it themselves and assigns the string.

## Light and dark

The GUI sets `data-bs-theme` on `<html>` from the `gui_theme` cookie (default `light`). Custom CSS is not split into two fields.

- Rules under `:root` or with no theme selector apply in **both** themes. That matches today's `PrimaryColor` injection.
- Theme-specific rules use `[data-bs-theme="light"]` and `[data-bs-theme="dark"]`.
- `@font-face` and `@import` stay at the top level of the blob. Do not auto-wrap the string in a selector.

Document this in `web/README.md` with the example above.

## Injection

Pass `CustomCSS` through `ResolvedBranding` and `Renderer.applyBranding` the same way as the other branding fields.

On `web.TemplateData`, store it as `template.CSS` so `html/template` does not escape `{` and `}`.

Emit this block only when the string is non-empty, **after** the existing branding CSS-variable `<style>` so custom rules win:

```html
{{if .CustomCSS}}
<style>
{{.CustomCSS}}
</style>
{{end}}
```

Pages that get the tag (same shells as current branding):

- `web/templates/layouts/base.tmpl`
- `web/templates/pages/login.tmpl`
- `web/templates/pages/2fa_verify.tmpl`

OIDC login, consent, logout, and error pages stay out of scope. Those use per-client login colors, not admin branding.

Do not change Content-Security-Policy. GUI `style-src` already allows `'unsafe-inline'`. `@font-face` URLs must be `'self'` or `https://cdn.jsdelivr.net` unless a later change widens `font-src`.

## Validation

`validateBranding` in `validate.go`, called from `ValidateConfig` / `app.New()`.

| Rule | On failure |
|------|------------|
| Length greater than 65536 bytes | Error naming `CustomCSS` and the limit |
| Case-insensitive substring `</style` | Error. Prevents breaking out of the `<style>` tag |
| Case-insensitive substring `<script` | Error |
| Case-insensitive substring `javascript:` | Error |

Empty string skips these checks.

Do not parse CSS. Invalid CSS is the consumer's problem. The browser ignores it.

## `cmd/api`

Map `ADMIN_BRANDING_CUSTOM_CSS` onto `Admin.Branding.CustomCSS`, same pattern as the other `ADMIN_BRANDING_*` keys. Document it in `docs/guides/ENV_VARIABLES.md` and comment it in `.env.example`.

Multiline CSS in a `.env` file is awkward. Library consumers should set the Go field. The env key exists so the reference app stays consistent.

## Tests

Add to `validate_test.go`:

- empty `CustomCSS` is valid
- a short stylesheet with `[data-bs-theme="dark"]` is valid
- 65537 bytes is rejected
- `</style`, `<script`, and `javascript:` (any case) are rejected

Add to `web/branding_test.go` or renderer tests:

- `ResolveBranding` copies `CustomCSS` through
- `applyBranding` sets `TemplateData.CustomCSS`

No screenshot or browser tests.

## Docs

- `web/README.md`: new section for `CustomCSS`, light/dark selectors, size/safety limits, CSP font note
- `docs/guides/ENV_VARIABLES.md` and `.env.example` as above
- `.claude/skills/go-core/references/admin-gui.md`: add `CustomCSS` to the `TemplateData` list

## Out of scope

- Database-backed branding or a GUI CSS editor
- File-path loading inside the module
- Favicon field
- Dedicated font config
- Separate light/dark config fields
- OIDC page injection
- CSP changes

## Files

- `config.go`
- `validate.go`, `validate_test.go`
- `web/branding.go`, `web/branding_test.go`
- `web/renderer.go`
- `web/templates/layouts/base.tmpl`
- `web/templates/pages/login.tmpl`
- `web/templates/pages/2fa_verify.tmpl`
- `cmd/api/main.go`
- `web/README.md`
- `docs/guides/ENV_VARIABLES.md`
- `.env.example`
- `.claude/skills/go-core/references/admin-gui.md`
