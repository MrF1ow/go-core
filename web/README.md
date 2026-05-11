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

## Logo

`LogoURL` accepts two formats:

- **URL** (`http://` or `https://`): used directly as `<img src>`.
- **File path**: the file is read at startup and served at `<AdminBasePath>/branding/logo` (default `/gui/branding/logo`). The file must exist when `app.New()` is called.

Supported formats: PNG, SVG, JPG, WebP — anything a browser `<img>` tag can render.

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
