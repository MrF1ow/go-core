# OIDC page admin branding

**Date:** 2026-08-21
**Approach:** Teach `Renderer.applyBranding` to fill empty keys on `gin.H`. OIDC login, consent, logout, and error pages then pick up admin `FaviconURL`, `CustomCSS`, `PrimaryColor`, and `BorderRadius` without new config fields. Client and app values already in the map win.

## Summary

Admin branding never reaches OIDC pages today. Those handlers pass `gin.H`. `applyBranding` only injects into `web.TemplateData`. The templates also have no favicon tag and no CustomCSS block.

Do not add fields to `AdminBrandingConfig`. Do not add per-client favicon or CSS. The deploy-level branding config is the fallback. Per-client `LogoURL` and `LoginPrimaryColor` stay the identity of the relying party.

## Why gin.H

Every OIDC HTML call site is `c.HTML(..., gin.H{...})` in `internal/oidc/handler.go`. `applyBranding` type-switches `TemplateData` and `*TemplateData` and returns any other type unchanged. Fix the renderer. Leave the OIDC handlers alone.

## Merge rules

On `gin.H` and `map[string]any`, set a branding key only when it is missing or an empty string. Never overwrite a non-empty value.

| Key | Source | Rule |
|-----|--------|------|
| `FaviconURL` | `ResolvedBranding.FaviconURL` | Fill if empty. Handlers never set this today. |
| `CustomCSS` | `template.CSS(ResolvedBranding.CustomCSS)` | Fill if empty. Same G203 note as `TemplateData`. |
| `PrimaryColor` | `ResolvedBranding.PrimaryColor` | Fill if empty. Client and app colors already in the map keep winning. |
| `BorderRadius` | `ResolvedBranding.BorderRadius` | Fill if empty. |

Do not copy these onto the map:

- `OrgName`. Titles stay `{{.AppName}}`. That is the tenant application, not the admin org.
- `LogoURL`. Do not substitute it for `ClientLogo`. The client logo is the relying party. The admin logo is the identity provider.
- `SidebarColor` / `SidebarTextColor` / `SecondaryColor`. No sidebar on these pages.

Empty admin branding still means empty keys. Templates then behave as they do on the admin GUI: no extra `<style>` for CustomCSS, shield data URI for favicon.

## Templates

Extract the favicon `<link>` into `web/templates/partials/favicon.tmpl` so the shield data URI is not copied four more times. Switch the three admin shells to `{{template "favicon" .}}` as well. Same markup as today:

```html
{{if .FaviconURL}}
    <link rel="icon" href="{{.FaviconURL}}">
{{else}}
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,...existing shield...">
{{end}}
```

Call it from:

- `web/templates/layouts/base.tmpl`
- `web/templates/pages/login.tmpl`
- `web/templates/pages/2fa_verify.tmpl`
- `web/templates/pages/oidc_login.tmpl`
- `web/templates/pages/oidc_consent.tmpl`
- `web/templates/pages/oidc_logout.tmpl`
- `web/templates/pages/oidc_error.tmpl`

On the four OIDC pages, after the existing `{{if .PrimaryColor}}` block, add BorderRadius and CustomCSS the same way `login.tmpl` does:

```html
{{if .BorderRadius}}
<style>
    :root { --bs-border-radius: {{.BorderRadius}}; }
</style>
{{end}}
{{if .CustomCSS}}
<style>
{{.CustomCSS}}
</style>
{{end}}
```

Keep the existing PrimaryColor block as-is. Do not set `type` on a custom favicon link. Do not mark URLs as `template.URL`.

OIDC theme (`light` / `dark` / `auto` from the client or app) is unchanged. CustomCSS that uses `[data-bs-theme="dark"]` still matches because those pages set `data-bs-theme` on `<html>`.

## CSP

No change. OIDC `img-src` is already `'self' data: https:`. `style-src` already allows `'unsafe-inline'`. `font-src` stays `'self' https://cdn.jsdelivr.net`.

## Validation and config

None. `validateBranding` already ran at `app.New()`.

## Tests

Add to `web/branding_test.go` (or renderer tests). Use `gin.H`, not `TemplateData`. That is the path OIDC actually uses.

- `applyBranding` on `gin.H` copies `FaviconURL` and `CustomCSS` when those keys are absent
- `applyBranding` does not overwrite a non-empty `PrimaryColor`
- `applyBranding` fills `PrimaryColor` when the map has `""`
- rendered `oidc_login` with `FaviconURL` set emits that href and not the shield data URI
- rendered `oidc_login` with empty branding still emits the shield data URI
- rendered `oidc_login` emits unescaped CustomCSS (same assertion as the admin login test)

Keep the existing `TemplateData` tests. They still cover the admin GUI.

No screenshot tests. No OIDC handler tests unless a handler actually has to change.

## Docs

- `web/README.md`: short "OIDC pages" section. Fallback list. What is not copied (`OrgName`, `LogoURL` as `ClientLogo`).
- `.claude/skills/go-core/references/admin-gui.md`: `applyBranding` also fills `gin.H`. OIDC pages listed next to the admin shells.
- `docs/README.md`: link this spec

## Out of scope

- New `AdminBrandingConfig` fields
- Per-client favicon or CustomCSS
- Using admin `LogoURL` as `ClientLogo`
- Replacing `AppName` with `OrgName`
- Changing `clientThemeWithOverride` / `appTheme`
- OIDC handler edits solely to pass branding
- `apple-touch-icon`
- `font-src` changes
- Database-backed branding

## Files

- `web/renderer.go`
- `web/branding_test.go`
- `web/templates/partials/favicon.tmpl`
- `web/templates/layouts/base.tmpl`
- `web/templates/pages/login.tmpl`
- `web/templates/pages/2fa_verify.tmpl`
- `web/templates/pages/oidc_login.tmpl`
- `web/templates/pages/oidc_consent.tmpl`
- `web/templates/pages/oidc_logout.tmpl`
- `web/templates/pages/oidc_error.tmpl`
- `web/README.md`
- `.claude/skills/go-core/references/admin-gui.md`
- `docs/README.md`
