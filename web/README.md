# Admin Dashboard Branding

Customize the admin dashboard appearance by setting fields on `AdminBrandingConfig` in your `core.Config`.

## Configuration

```go
cfg := core.Config{
    Admin: core.AdminConfig{
        Branding: core.AdminBrandingConfig{
            OrgName:      "Acme Corp",
            LogoURL:      "https://acme.com/logo.svg",
            PrimaryColor: "#4f46e5",
            BorderRadius: "0.75rem",
            SidebarColor: "#1a1a2e",
        },
    },
}
```

All fields are optional. Zero values produce the default Bootstrap look.

## Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `OrgName` | `string` | `"Auth API"` | Organization name shown on the login page and sidebar header. |
| `LogoURL` | `string` | Shield icon | URL or file path to your logo. Shown at 64px on the login page and 28px in the sidebar. |
| `PrimaryColor` | `string` | Bootstrap blue (`#0d6efd`) | Hex color (`#RGB` or `#RRGGBB`). Overrides buttons, links, and active states. |
| `SecondaryColor` | `string` | Bootstrap default | Hex color. Overrides the Bootstrap secondary color. |
| `BorderRadius` | `string` | Bootstrap default | CSS length (e.g. `"0.5rem"`, `"0"`, `"8px"`). Controls roundness of cards, buttons, inputs. |
| `SidebarColor` | `string` | `#212529` light / `#101214` dark | Hex color for sidebar background. Applies to both light and dark themes. |
| `SidebarTextColor` | `string` | Auto-derived | Hex color for sidebar text. When empty, automatically picks `#ffffff` (dark bg) or `#212529` (light bg) based on sidebar background luminance. |
| `CustomCSS` | `string` | off | Raw CSS injected after the branding `<style>` block. Empty string means no extra tag. |
| `FaviconURL` | `string` | Shield SVG data URI | URL or file path for the tab icon. Empty keeps the built-in shield. |

## Logo

`LogoURL` accepts two formats:

- **URL** (`http://` or `https://`): used directly as `<img src>`.
- **File path**: the file is read at startup and served at `<AdminBasePath>/branding/logo` (default `/gui/branding/logo`). The file must exist when `app.New()` is called.

Supported formats: PNG, SVG, JPG, WebP — anything a browser `<img>` tag can render.

GUI `img-src` is `'self' data: https:`. An `https://` `LogoURL` loads. An `http://` `LogoURL` is accepted by validation but blocked by CSP.

## Favicon

`FaviconURL` uses the same URL-or-file split as `LogoURL`. Empty keeps the built-in shield SVG data URI. The field does not fall back to `LogoURL`.

- **URL** (`http://` or `https://`): used as `<link rel="icon" href="...">`.
- **File path**: the file is read at startup and served at `<AdminBasePath>/branding/favicon` (default `/gui/branding/favicon`). The file must exist when `app.New()` is called.

ICO, PNG, SVG, GIF, WebP, and JPEG are all valid. File size is capped at 1 MiB, same as `LogoURL`.

Remote `https://` favicons are allowed by GUI CSP. Remote `http://` values are not.

## Sidebar Text Auto-Derivation

When `SidebarColor` is set but `SidebarTextColor` is empty, the text color is computed automatically using WCAG relative luminance:

- Luminance > 0.5 → dark text (`#212529`)
- Luminance ≤ 0.5 → light text (`#ffffff`)

Set `SidebarTextColor` explicitly to override this behavior.

## CSS Variables Overridden

When branding is configured, these Bootstrap CSS variables are overridden via inline `<style>` blocks:

- `--bs-primary` (from `PrimaryColor`)
- `--bs-secondary` (from `SecondaryColor`)
- `--bs-border-radius` (from `BorderRadius`)
- `.btn-primary` component variables (hover, active states derived via `color-mix`)
- `.sidebar` background and nav-link colors (from `SidebarColor` / `SidebarTextColor`)

## Custom CSS

`CustomCSS` is a raw stylesheet string. The module does not read a file. If you keep CSS on disk, read it yourself and assign the string.

Light and dark live in the same blob. The GUI sets `data-bs-theme` on `<html>` from the `gui_theme` cookie (default `light`). Rules under `:root` or with no theme selector apply in both themes. Theme-specific rules use `[data-bs-theme="light"]` and `[data-bs-theme="dark"]`. `@font-face` and `@import` stay at the top of the blob. The library does not wrap the string in a selector.

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

Limits, checked in `ValidateConfig`:

- Maximum 65536 bytes (64KiB).
- Case-insensitive substrings `</style`, `<script`, and `javascript:` are rejected so the blob cannot break out of the `<style>` tag.

Invalid CSS is not parsed. The browser ignores it.

Content-Security-Policy is unchanged. GUI `style-src` already allows `'unsafe-inline'`. `@font-face` URLs must be `'self'` or `https://cdn.jsdelivr.net` unless a later change widens `font-src`.
