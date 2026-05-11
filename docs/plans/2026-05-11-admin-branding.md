# Admin Dashboard Branding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow consumers to customize the admin dashboard appearance (logo, colors, border radius, sidebar) via `AdminBrandingConfig` fields in the config struct.

**Architecture:** Add `AdminBrandingConfig` to the existing config, resolve branding values (including sidebar text auto-derivation) at init time in a new `web/branding.go`, store resolved values on the `Renderer`, and auto-inject them into every `TemplateData`. Templates use conditional CSS variable overrides — same pattern as existing OIDC page theming.

**Tech Stack:** Go, Bootstrap 5.3 CSS variables, Go html/template, Gin

---

### Task 1: Add `AdminBrandingConfig` Struct and Validation

**Files:**
- Modify: `config.go:100-108` (AdminConfig struct)
- Modify: `validate.go` (add branding validation)
- Modify: `validate_test.go` (add branding validation tests)

- [ ] **Step 1: Write failing tests for branding validation**

Add to `validate_test.go`:

```go
func TestValidateConfig_InvalidPrimaryColor(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.PrimaryColor = "not-a-color"
	assertValidationError(t, cfg, "PrimaryColor")
}

func TestValidateConfig_ValidPrimaryColor(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.PrimaryColor = "#4f46e5"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_ValidShortHexColor(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.PrimaryColor = "#f00"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_InvalidSecondaryColor(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.SecondaryColor = "rgb(0,0,0)"
	assertValidationError(t, cfg, "SecondaryColor")
}

func TestValidateConfig_InvalidSidebarColor(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.SidebarColor = "#GGHHII"
	assertValidationError(t, cfg, "SidebarColor")
}

func TestValidateConfig_InvalidSidebarTextColor(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.SidebarTextColor = "white"
	assertValidationError(t, cfg, "SidebarTextColor")
}

func TestValidateConfig_InvalidBorderRadius(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.BorderRadius = "abc"
	assertValidationError(t, cfg, "BorderRadius")
}

func TestValidateConfig_ValidBorderRadius(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.BorderRadius = "0.5rem"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_ZeroBorderRadius(t *testing.T) {
	cfg := validConfig()
	cfg.Admin.Branding.BorderRadius = "0"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_EmptyBrandingIsValid(t *testing.T) {
	cfg := validConfig()
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error with empty branding, got: %v", err)
	}
}
```

Helper function (add near top of `validate_test.go` if not already present):

```go
func validConfig() Config {
	return Config{
		Database: DatabaseConfig{
			Host:   "localhost",
			Port:   5432,
			DBName: "test",
			User:   "postgres",
		},
		JWT: JWTConfig{
			Secret: "this-is-a-secret-that-is-at-least-32-chars",
		},
	}
}

func assertValidationError(t *testing.T, cfg Config, fieldName string) {
	t.Helper()
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatalf("expected validation error for %s, got nil", fieldName)
	}
	if !strings.Contains(err.Error(), fieldName) {
		t.Fatalf("expected error to mention %s, got: %v", fieldName, err)
	}
}
```

Note: add `"strings"` to test file imports if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./. -run TestValidateConfig_Invalid -count=1`
Expected: compilation errors — `AdminBrandingConfig` and `Branding` field don't exist yet.

- [ ] **Step 3: Add `AdminBrandingConfig` struct to `config.go`**

In `config.go`, add the new struct before `AdminConfig` and add the `Branding` field:

```go
// AdminBrandingConfig customizes the admin dashboard appearance.
// All fields are optional — zero values produce the default Bootstrap look.
type AdminBrandingConfig struct {
	OrgName          string
	LogoURL          string
	PrimaryColor     string
	SecondaryColor   string
	BorderRadius     string
	SidebarColor     string
	SidebarTextColor string
}
```

Add the `Branding` field to the existing `AdminConfig` struct:

```go
type AdminConfig struct {
	APIKey     string
	Email      string
	SessionTTL time.Duration
	BaseURL    string
	Branding   AdminBrandingConfig
}
```

- [ ] **Step 4: Add branding validation to `validate.go`**

Add a helper and call it at the end of `ValidateConfig`:

```go
import "regexp"

var (
	hexColorRegex   = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	borderRadiusRegex = regexp.MustCompile(`^(0|[0-9]+(\.[0-9]+)?(px|rem|em))$`)
)

func validateBranding(b AdminBrandingConfig) error {
	if b.PrimaryColor != "" && !hexColorRegex.MatchString(b.PrimaryColor) {
		return fmt.Errorf("core: Config.Admin.Branding.PrimaryColor must be a valid hex color (#RGB or #RRGGBB)")
	}
	if b.SecondaryColor != "" && !hexColorRegex.MatchString(b.SecondaryColor) {
		return fmt.Errorf("core: Config.Admin.Branding.SecondaryColor must be a valid hex color (#RGB or #RRGGBB)")
	}
	if b.SidebarColor != "" && !hexColorRegex.MatchString(b.SidebarColor) {
		return fmt.Errorf("core: Config.Admin.Branding.SidebarColor must be a valid hex color (#RGB or #RRGGBB)")
	}
	if b.SidebarTextColor != "" && !hexColorRegex.MatchString(b.SidebarTextColor) {
		return fmt.Errorf("core: Config.Admin.Branding.SidebarTextColor must be a valid hex color (#RGB or #RRGGBB)")
	}
	if b.BorderRadius != "" && !borderRadiusRegex.MatchString(b.BorderRadius) {
		return fmt.Errorf("core: Config.Admin.Branding.BorderRadius must be a valid CSS length (e.g. \"0.5rem\", \"0\", \"8px\")")
	}
	if b.LogoURL != "" {
		isURL := strings.HasPrefix(b.LogoURL, "http://") || strings.HasPrefix(b.LogoURL, "https://")
		if !isURL {
			if _, err := os.Stat(b.LogoURL); err != nil {
				return fmt.Errorf("core: Config.Admin.Branding.LogoURL file not found: %s", b.LogoURL)
			}
		}
	}
	return nil
}
```

Add `"os"` and `"strings"` to the imports of `validate.go`. Add `"regexp"` to imports.

At the end of `ValidateConfig`, before the final `return nil`:

```go
if err := validateBranding(cfg.Admin.Branding); err != nil {
	return err
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -v ./. -run TestValidateConfig -count=1`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add config.go validate.go validate_test.go
git commit -m "feat(admin): add AdminBrandingConfig struct with validation"
```

---

### Task 2: Create `web/branding.go` — Luminance and Resolution

**Files:**
- Create: `web/branding.go`
- Create: `web/branding_test.go`

- [ ] **Step 1: Write failing tests for hex parsing and luminance**

Create `web/branding_test.go`:

```go
package web

import (
	"testing"
)

func TestParseHexColor_SixDigit(t *testing.T) {
	r, g, b, err := ParseHexColor("#ff8800")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 255 || g != 136 || b != 0 {
		t.Fatalf("expected (255, 136, 0), got (%d, %d, %d)", r, g, b)
	}
}

func TestParseHexColor_ThreeDigit(t *testing.T) {
	r, g, b, err := ParseHexColor("#f00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 255 || g != 0 || b != 0 {
		t.Fatalf("expected (255, 0, 0), got (%d, %d, %d)", r, g, b)
	}
}

func TestParseHexColor_Invalid(t *testing.T) {
	_, _, _, err := ParseHexColor("not-a-color")
	if err == nil {
		t.Fatal("expected error for invalid hex color")
	}
}

func TestParseHexColor_CaseInsensitive(t *testing.T) {
	r, g, b, err := ParseHexColor("#FF8800")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 255 || g != 136 || b != 0 {
		t.Fatalf("expected (255, 136, 0), got (%d, %d, %d)", r, g, b)
	}
}

func TestRelativeLuminance_White(t *testing.T) {
	l := RelativeLuminance(255, 255, 255)
	if l < 0.99 {
		t.Fatalf("expected ~1.0 for white, got %f", l)
	}
}

func TestRelativeLuminance_Black(t *testing.T) {
	l := RelativeLuminance(0, 0, 0)
	if l > 0.01 {
		t.Fatalf("expected ~0.0 for black, got %f", l)
	}
}

func TestAutoSidebarTextColor_DarkBackground(t *testing.T) {
	color := AutoSidebarTextColor("#1a1a2e")
	if color != "#ffffff" {
		t.Fatalf("expected #ffffff for dark bg, got %s", color)
	}
}

func TestAutoSidebarTextColor_LightBackground(t *testing.T) {
	color := AutoSidebarTextColor("#f0f0f0")
	if color != "#212529" {
		t.Fatalf("expected #212529 for light bg, got %s", color)
	}
}

func TestAutoSidebarTextColor_InvalidFallsBackToWhite(t *testing.T) {
	color := AutoSidebarTextColor("bad")
	if color != "#ffffff" {
		t.Fatalf("expected #ffffff fallback, got %s", color)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./web -run TestParseHexColor -count=1`
Expected: compilation errors — functions don't exist yet.

- [ ] **Step 3: Implement `web/branding.go`**

Create `web/branding.go`:

```go
package web

import (
	"fmt"
	"math"
	"strconv"
)

// ParseHexColor parses a CSS hex color string (#RGB or #RRGGBB) into RGB components.
func ParseHexColor(hex string) (r, g, b uint8, err error) {
	if len(hex) == 0 || hex[0] != '#' {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %q", hex)
	}
	hex = hex[1:]

	switch len(hex) {
	case 3:
		rr, err1 := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		gg, err2 := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		bb, err3 := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, fmt.Errorf("invalid hex color: #%s", hex)
		}
		return uint8(rr), uint8(gg), uint8(bb), nil
	case 6:
		rr, err1 := strconv.ParseUint(hex[0:2], 16, 8)
		gg, err2 := strconv.ParseUint(hex[2:4], 16, 8)
		bb, err3 := strconv.ParseUint(hex[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, fmt.Errorf("invalid hex color: #%s", hex)
		}
		return uint8(rr), uint8(gg), uint8(bb), nil
	default:
		return 0, 0, 0, fmt.Errorf("invalid hex color: #%s", hex)
	}
}

// RelativeLuminance computes the WCAG 2.0 relative luminance of an sRGB color.
// Returns a value between 0.0 (black) and 1.0 (white).
func RelativeLuminance(r, g, b uint8) float64 {
	rl := linearize(float64(r) / 255.0)
	gl := linearize(float64(g) / 255.0)
	bl := linearize(float64(b) / 255.0)
	return 0.2126*rl + 0.7152*gl + 0.0722*bl
}

func linearize(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// AutoSidebarTextColor returns "#ffffff" or "#212529" based on the luminance
// of the given hex background color. Falls back to "#ffffff" on parse error.
func AutoSidebarTextColor(sidebarHex string) string {
	r, g, b, err := ParseHexColor(sidebarHex)
	if err != nil {
		return "#ffffff"
	}
	if RelativeLuminance(r, g, b) > 0.5 {
		return "#212529"
	}
	return "#ffffff"
}

// ResolvedBranding holds the final computed branding values ready for template injection.
type ResolvedBranding struct {
	OrgName          string
	LogoURL          string
	PrimaryColor     string
	SecondaryColor   string
	BorderRadius     string
	SidebarColor     string
	SidebarTextColor string
}

// ResolveBranding takes raw config values and computes derived fields.
// logoServeURL is the internal URL to use when LogoURL is a file path
// (e.g., "/gui/branding/logo"). Pass empty string if LogoURL is already a URL.
func ResolveBranding(orgName, logoURL, primaryColor, secondaryColor, borderRadius, sidebarColor, sidebarTextColor, logoServeURL string) ResolvedBranding {
	resolved := ResolvedBranding{
		OrgName:        orgName,
		LogoURL:        logoURL,
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		BorderRadius:   borderRadius,
		SidebarColor:   sidebarColor,
	}

	if logoServeURL != "" {
		resolved.LogoURL = logoServeURL
	}

	if sidebarTextColor != "" {
		resolved.SidebarTextColor = sidebarTextColor
	} else if sidebarColor != "" {
		resolved.SidebarTextColor = AutoSidebarTextColor(sidebarColor)
	}

	return resolved
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v ./web -run "TestParseHex|TestRelativeLum|TestAutoSidebar" -count=1`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add web/branding.go web/branding_test.go
git commit -m "feat(admin): add branding utilities — hex parsing, luminance, auto-derive"
```

---

### Task 3: Wire Branding Into Renderer and TemplateData

**Files:**
- Modify: `web/renderer.go:25-55` (TemplateData struct)
- Modify: `web/renderer.go:57-61` (Renderer struct)
- Modify: `web/renderer.go:78-95` (Instance method)

- [ ] **Step 1: Add branding fields to `TemplateData`**

In `web/renderer.go`, add these fields to the `TemplateData` struct after the `Theme` field (line ~51):

```go
	// Branding (auto-populated by Renderer from resolved config)
	OrgName          string
	LogoURL          string
	PrimaryColor     string
	SecondaryColor   string
	BorderRadius     string
	SidebarColor     string
	SidebarTextColor string
```

- [ ] **Step 2: Add branding storage to Renderer struct**

In `web/renderer.go`, add a field to the `Renderer` struct:

```go
type Renderer struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
	branding  ResolvedBranding
}
```

- [ ] **Step 3: Add `SetBranding` method**

Add after the `NewRenderer` function:

```go
// SetBranding stores the resolved branding config. Call once during initialization.
func (r *Renderer) SetBranding(b ResolvedBranding) {
	r.branding = b
}
```

- [ ] **Step 4: Modify `Instance` to auto-populate branding**

Replace the `Instance` method body to inject branding into `TemplateData`. Note: GUI handlers pass `TemplateData` by value (not pointer), so we must handle the value type, mutate it, and pass the modified copy:

```go
func (r *Renderer) Instance(name string, data interface{}) render.Render {
	tmpl, ok := r.templates[name]
	if !ok {
		return &HTMLRender{
			Template: nil,
			Name:     name,
			Data:     data,
		}
	}

	data = r.applyBranding(data)

	return &HTMLRender{
		Template: tmpl,
		Name:     name,
		Data:     data,
	}
}

// applyBranding injects resolved branding fields into TemplateData.
// Handles both value and pointer types since handlers pass by value.
func (r *Renderer) applyBranding(data interface{}) interface{} {
	inject := func(td *TemplateData) {
		td.OrgName = r.branding.OrgName
		td.LogoURL = r.branding.LogoURL
		td.PrimaryColor = r.branding.PrimaryColor
		td.SecondaryColor = r.branding.SecondaryColor
		td.BorderRadius = r.branding.BorderRadius
		td.SidebarColor = r.branding.SidebarColor
		td.SidebarTextColor = r.branding.SidebarTextColor
	}

	switch td := data.(type) {
	case TemplateData:
		inject(&td)
		return td
	case *TemplateData:
		inject(td)
		return td
	default:
		return data
	}
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./web/...`
Expected: compiles successfully.

- [ ] **Step 6: Commit**

```bash
git add web/renderer.go
git commit -m "feat(admin): wire branding into Renderer and TemplateData"
```

---

### Task 4: Wire Branding Into `coreapp.RegisterRoutes`

**Files:**
- Modify: `internal/coreapp/app.go:397-410` (RegisterRoutes — renderer init)
- Modify: `internal/coreapp/app.go:667-670` (GUI routes — logo serving)

- [ ] **Step 1: Call `SetBranding` after renderer creation**

In `internal/coreapp/app.go`, in the `RegisterRoutes` method, after `r.HTMLRender = renderer` (around line 408), add:

```go
	// Apply admin branding to renderer
	b := cfg.Admin.Branding
	isFileURL := b.LogoURL != "" &&
		!strings.HasPrefix(b.LogoURL, "http://") &&
		!strings.HasPrefix(b.LogoURL, "https://")
	logoServeURL := ""
	if isFileURL {
		logoServeURL = "/gui/branding/logo"
	}
	renderer.SetBranding(web.ResolveBranding(
		b.OrgName,
		b.LogoURL,
		b.PrimaryColor,
		b.SecondaryColor,
		b.BorderRadius,
		b.SidebarColor,
		b.SidebarTextColor,
		logoServeURL,
	))
```

Add `"strings"` to the imports if not already present. Add `"io"` and `"net/http"` imports (needed for logo route below — check if already imported).

- [ ] **Step 2: Add logo file serving route**

In the GUI routes section, after the static assets line (`gui.StaticFS("/static", static.HTTPFileSystem())`), add:

```go
		// Serve branding logo from local file (if configured)
		if isFileURL {
			logoData, err := os.ReadFile(b.LogoURL)
			if err != nil {
				log.Fatalf("Failed to read branding logo file: %v", err)
			}
			logoContentType := http.DetectContentType(logoData)
			gui.GET("/branding/logo", func(c *gin.Context) {
				c.Data(http.StatusOK, logoContentType, logoData)
			})
		}
```

Move the `isFileURL` and `logoServeURL` variable declarations above the `gui` group so they're accessible in both the renderer setup and route registration. Alternatively, declare them once before both blocks.

Add `"os"` to the imports if not already present.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/coreapp/...`
Expected: compiles successfully.

- [ ] **Step 4: Commit**

```bash
git add internal/coreapp/app.go
git commit -m "feat(admin): wire branding resolution and logo route into RegisterRoutes"
```

---

### Task 5: Update `base.tmpl` — CSS Injection and Sidebar Branding

**Files:**
- Modify: `web/templates/layouts/base.tmpl:118-130` (after style block, before body)
- Modify: `web/templates/layouts/base.tmpl:122-129` (sidebar header)

- [ ] **Step 1: Add CSS variable injection block**

In `base.tmpl`, after the closing `</style>` tag of the existing style block (after the dark mode table overrides, around line 118) and before `</head>`, add:

```html
    {{if or .PrimaryColor .SecondaryColor .BorderRadius .SidebarColor}}
    <style>
        {{if or .PrimaryColor .SecondaryColor .BorderRadius}}
        :root {
            {{if .PrimaryColor}}--bs-primary: {{.PrimaryColor}};{{end}}
            {{if .SecondaryColor}}--bs-secondary: {{.SecondaryColor}};{{end}}
            {{if .BorderRadius}}--bs-border-radius: {{.BorderRadius}};{{end}}
        }
        {{end}}
        {{if .PrimaryColor}}
        .btn-primary {
            --bs-btn-bg: {{.PrimaryColor}};
            --bs-btn-border-color: {{.PrimaryColor}};
            --bs-btn-hover-bg: color-mix(in srgb, {{.PrimaryColor}} 85%, black);
            --bs-btn-hover-border-color: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
            --bs-btn-active-bg: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
            --bs-btn-active-border-color: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
        }
        .text-primary { color: {{.PrimaryColor}} !important; }
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

- [ ] **Step 2: Update sidebar header with logo and org name**

Find the sidebar header section (around lines 122-129):

```html
        <div class="p-3">
            <a href="/gui/" class="text-white text-decoration-none d-flex align-items-center sidebar-link"
               data-page="dashboard"
               hx-get="/gui/" hx-target="#page-content" hx-select="#page-content" hx-swap="outerHTML show:no-scroll" hx-push-url="true">
                <i class="bi bi-shield-lock fs-4 me-2"></i>
                <span class="fs-5 fw-semibold">Auth API</span>
            </a>
        </div>
```

Replace the icon and text lines with:

```html
        <div class="p-3">
            <a href="/gui/" class="text-white text-decoration-none d-flex align-items-center sidebar-link"
               data-page="dashboard"
               hx-get="/gui/" hx-target="#page-content" hx-select="#page-content" hx-swap="outerHTML show:no-scroll" hx-push-url="true">
                {{if .LogoURL}}<img src="{{.LogoURL}}" alt="{{if .OrgName}}{{.OrgName}}{{else}}Auth API{{end}}" style="height:28px;width:28px;object-fit:contain;" class="me-2">{{else}}<i class="bi bi-shield-lock fs-4 me-2"></i>{{end}}
                <span class="fs-5 fw-semibold">{{if .OrgName}}{{.OrgName}}{{else}}Auth API{{end}}</span>
            </a>
        </div>
```

- [ ] **Step 3: Verify template parses**

Run: `go test -v ./web -run TestNewRenderer -count=1` (if such a test exists) or `go build ./...`
Expected: compiles and templates parse without error.

- [ ] **Step 4: Commit**

```bash
git add web/templates/layouts/base.tmpl
git commit -m "feat(admin): add branding CSS injection and logo/name to sidebar"
```

---

### Task 6: Update `login.tmpl` — Branding

**Files:**
- Modify: `web/templates/pages/login.tmpl`

- [ ] **Step 1: Add CSS variable injection to login page**

In `login.tmpl`, after the existing `<style>` block (after the `.divider` styles, before `</head>`), add:

```html
    {{if or .PrimaryColor .BorderRadius}}
    <style>
        {{if .PrimaryColor}}
        :root {
            --bs-primary: {{.PrimaryColor}};
        }
        .btn-primary {
            --bs-btn-bg: {{.PrimaryColor}};
            --bs-btn-border-color: {{.PrimaryColor}};
            --bs-btn-hover-bg: color-mix(in srgb, {{.PrimaryColor}} 85%, black);
            --bs-btn-hover-border-color: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
            --bs-btn-active-bg: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
            --bs-btn-active-border-color: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
        }
        .text-primary { color: {{.PrimaryColor}} !important; }
        {{end}}
        {{if .BorderRadius}}
        :root { --bs-border-radius: {{.BorderRadius}}; }
        {{end}}
    </style>
    {{end}}
```

- [ ] **Step 2: Replace shield icon with conditional logo**

Find the branding header (around line 61-64):

```html
            <i class="bi bi-shield-lock text-primary" style="font-size: 3rem;"></i>
            <h3 class="mt-2 fw-bold">Auth API</h3>
            <p class="text-muted">Admin Panel</p>
```

Replace with:

```html
            {{if .LogoURL}}<img src="{{.LogoURL}}" alt="{{if .OrgName}}{{.OrgName}}{{else}}Auth API{{end}}" style="height:64px;width:64px;object-fit:contain;">{{else}}<i class="bi bi-shield-lock text-primary" style="font-size: 3rem;"></i>{{end}}
            <h3 class="mt-2 fw-bold">{{if .OrgName}}{{.OrgName}}{{else}}Auth API{{end}}</h3>
            <p class="text-muted">Admin Panel</p>
```

- [ ] **Step 3: Verify template parses**

Run: `go build ./...`
Expected: compiles successfully.

- [ ] **Step 4: Commit**

```bash
git add web/templates/pages/login.tmpl
git commit -m "feat(admin): add branding to login page — logo, colors, org name"
```

---

### Task 7: Update `2fa_verify.tmpl` — Branding

**Files:**
- Modify: `web/templates/pages/2fa_verify.tmpl`

- [ ] **Step 1: Add CSS variable injection**

In `2fa_verify.tmpl`, after the existing `<style>` block (after `.code-input` styles, before `</head>`), add:

```html
    {{if or .PrimaryColor .BorderRadius}}
    <style>
        {{if .PrimaryColor}}
        :root {
            --bs-primary: {{.PrimaryColor}};
        }
        .btn-primary {
            --bs-btn-bg: {{.PrimaryColor}};
            --bs-btn-border-color: {{.PrimaryColor}};
            --bs-btn-hover-bg: color-mix(in srgb, {{.PrimaryColor}} 85%, black);
            --bs-btn-hover-border-color: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
            --bs-btn-active-bg: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
            --bs-btn-active-border-color: color-mix(in srgb, {{.PrimaryColor}} 75%, black);
        }
        .text-primary { color: {{.PrimaryColor}} !important; }
        {{end}}
        {{if .BorderRadius}}
        :root { --bs-border-radius: {{.BorderRadius}}; }
        {{end}}
    </style>
    {{end}}
```

- [ ] **Step 2: Replace shield icon with conditional logo**

Find the branding area (around line 48-50):

```html
            <i class="bi bi-shield-check text-primary" style="font-size: 3rem;"></i>
            <h3 class="mt-2 fw-bold">Two-Factor Verification</h3>
```

Replace the icon line with:

```html
            {{if .LogoURL}}<img src="{{.LogoURL}}" alt="{{if .OrgName}}{{.OrgName}}{{else}}Auth API{{end}}" style="height:64px;width:64px;object-fit:contain;">{{else}}<i class="bi bi-shield-check text-primary" style="font-size: 3rem;"></i>{{end}}
```

Note: keep "Two-Factor Verification" as the heading text — this is a functional label, not branding.

- [ ] **Step 3: Verify template parses**

Run: `go build ./...`
Expected: compiles successfully.

- [ ] **Step 4: Commit**

```bash
git add web/templates/pages/2fa_verify.tmpl
git commit -m "feat(admin): add branding to 2FA verification page"
```

---

### Task 8: Create `web/README.md` Documentation

**Files:**
- Create: `web/README.md`

- [ ] **Step 1: Write `web/README.md`**

```markdown
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
- **File path**: the file is read at startup and served at `/gui/branding/logo`. The file must exist when `app.New()` is called.

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
```

- [ ] **Step 2: Commit**

```bash
git add web/README.md
git commit -m "docs(admin): add web/README.md documenting admin branding config"
```

---

### Task 9: Update README.md and CLAUDE.md References

**Files:**
- Modify: `README.md` (config table, around line 77)
- Modify: `CLAUDE.md` (web/ directory description, around line 80)

- [ ] **Step 1: Update README.md config table**

Find this line in the config table:

```markdown
| `Admin` | Admin GUI settings and API key for admin endpoints | Disabled. |
```

Replace with:

```markdown
| `Admin` | Admin GUI settings, API key, and [branding](web/README.md) | Disabled. |
```

- [ ] **Step 2: Update CLAUDE.md web/ description**

Find this line in CLAUDE.md:

```markdown
- `web/` — embedded HTMX templates and static assets for admin GUI
```

Replace with:

```markdown
- `web/` — embedded HTMX templates, static assets, and [branding](web/README.md) for admin GUI
```

- [ ] **Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: reference admin branding docs in README and CLAUDE.md"
```

---

### Task 10: Run Full CI Pipeline

**Files:** None (verification only)

- [ ] **Step 1: Format code**

Run: `make fmt`
Expected: no changes or auto-formatted.

- [ ] **Step 2: Lint**

Run: `make lint`
Expected: passes with no errors.

- [ ] **Step 3: Run all tests**

Run: `make test`
Expected: all tests pass, including new branding validation and utility tests.

- [ ] **Step 4: Security scan**

Run: `make security`
Expected: no new findings.

- [ ] **Step 5: Build**

Run: `make build-prod`
Expected: compiles successfully.

- [ ] **Step 6: Fix any issues found, then commit**

If any step above produces failures, fix them and commit fixes before proceeding.
