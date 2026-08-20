package web

import (
	"html/template"
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
	got := ResolveBranding("", "", "", "", "", "", "", css, "")
	if got.CustomCSS != css {
		t.Fatalf("expected CustomCSS %q, got %q", css, got.CustomCSS)
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
