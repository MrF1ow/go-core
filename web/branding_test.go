package web

import (
	"html/template"
	"net/http/httptest"
	"strings"
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

func TestResolveBranding_CopiesCustomCSS(t *testing.T) {
	css := `[data-bs-theme="dark"] { --bs-body-bg: #121212; }`
	got := ResolveBranding(BrandingInput{CustomCSS: css})
	if got.CustomCSS != css {
		t.Fatalf("expected CustomCSS %q, got %q", css, got.CustomCSS)
	}
}

func TestResolveBranding_CopiesFaviconURL(t *testing.T) {
	url := "https://example.com/favicon.ico"
	got := ResolveBranding(BrandingInput{FaviconURL: url})
	if got.FaviconURL != url {
		t.Fatalf("expected FaviconURL %q, got %q", url, got.FaviconURL)
	}
}

func TestResolveBranding_FaviconServeURLWins(t *testing.T) {
	got := ResolveBranding(BrandingInput{
		FaviconURL:      "https://example.com/favicon.ico",
		FaviconServeURL: "/gui/branding/favicon",
	})
	if got.FaviconURL != "/gui/branding/favicon" {
		t.Fatalf("expected FaviconURL %q, got %q", "/gui/branding/favicon", got.FaviconURL)
	}
}

func TestApplyBranding_SetsCustomCSS(t *testing.T) {
	css := "body { font-family: system-ui; }"
	r := &Renderer{
		branding: ResolvedBranding{CustomCSS: css},
	}
	data := r.applyBranding(TemplateData{}).(TemplateData)
	if data.CustomCSS != template.CSS(css) {
		t.Fatalf("expected TemplateData.CustomCSS %q, got %q", css, data.CustomCSS)
	}
}

func TestApplyBranding_SetsFaviconURL(t *testing.T) {
	url := "https://example.com/favicon.ico"
	r := &Renderer{
		branding: ResolvedBranding{FaviconURL: url},
	}
	data := r.applyBranding(TemplateData{}).(TemplateData)
	if data.FaviconURL != url {
		t.Fatalf("expected TemplateData.FaviconURL %q, got %q", url, data.FaviconURL)
	}
}

func TestLoginTemplate_EmitsUnescapedCustomCSS(t *testing.T) {
	r, err := NewRenderer("/gui")
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	css := `[data-bs-theme="dark"] .card > .body { color: #fff; }`
	r.SetBranding(ResolvedBranding{CustomCSS: css})
	rec := httptest.NewRecorder()
	if err := r.Instance("login", TemplateData{}).Render(rec); err != nil {
		t.Fatalf("render login: %v", err)
	}
	out := rec.Body.String()
	if !strings.Contains(out, css) {
		t.Fatalf("rendered login missing CustomCSS %q", css)
	}
	if strings.Contains(out, ".card &gt; .body") {
		t.Fatal("CustomCSS child combinator was HTML-escaped")
	}
}

const shieldFaviconURI = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Cpath d='M16 1L3 7v8c0 8 5 14 13 16 8-2 13-8 13-16V7Z' fill='%23212529'/%3E%3Crect x='12' y='14' width='8' height='7' rx='1.5' fill='%23fff'/%3E%3Cpath d='M13.5 14v-2.5a2.5 2.5 0 0 1 5 0V14' fill='none' stroke='%23fff' stroke-width='1.8'/%3E%3C/svg%3E"

func TestLoginTemplate_EmitsCustomFavicon(t *testing.T) {
	r, err := NewRenderer("/gui")
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	url := "https://example.com/favicon.ico"
	r.SetBranding(ResolvedBranding{FaviconURL: url})
	rec := httptest.NewRecorder()
	if err := r.Instance("login", TemplateData{}).Render(rec); err != nil {
		t.Fatalf("render login: %v", err)
	}
	out := rec.Body.String()
	if !strings.Contains(out, `href="`+url+`"`) {
		t.Fatalf("rendered login missing favicon href %q", url)
	}
	if strings.Contains(out, shieldFaviconURI) {
		t.Fatal("rendered login still emits the default shield data URI")
	}
}

func TestLoginTemplate_EmitsShieldFaviconWhenEmpty(t *testing.T) {
	r, err := NewRenderer("/gui")
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	rec := httptest.NewRecorder()
	if err := r.Instance("login", TemplateData{}).Render(rec); err != nil {
		t.Fatalf("render login: %v", err)
	}
	out := rec.Body.String()
	if !strings.Contains(out, shieldFaviconURI) {
		t.Fatal("rendered login missing default shield data URI")
	}
}
