package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfig_ValidConfig(t *testing.T) {
	cfg := validTestConfig()
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_MissingDatabaseHost(t *testing.T) {
	cfg := validTestConfig()
	cfg.Database.Host = ""
	assertValidationError(t, cfg, "Config.Database.Host is required")
}

func TestValidateConfig_MissingDatabasePort(t *testing.T) {
	cfg := validTestConfig()
	cfg.Database.Port = 0
	assertValidationError(t, cfg, "Config.Database.Port must be > 0")
}

func TestValidateConfig_MissingDatabaseName(t *testing.T) {
	cfg := validTestConfig()
	cfg.Database.DBName = ""
	assertValidationError(t, cfg, "Config.Database.DBName is required")
}

func TestValidateConfig_MissingDatabaseUser(t *testing.T) {
	cfg := validTestConfig()
	cfg.Database.User = ""
	assertValidationError(t, cfg, "Config.Database.User is required")
}

func TestValidateConfig_MissingJWTSecret(t *testing.T) {
	cfg := validTestConfig()
	cfg.JWT.Secret = ""
	assertValidationError(t, cfg, "Config.JWT.Secret is required")
}

func TestValidateConfig_ShortJWTSecret(t *testing.T) {
	cfg := validTestConfig()
	cfg.JWT.Secret = "tooshort"
	assertValidationError(t, cfg, "Config.JWT.Secret must be at least 32 characters")
}

func TestValidateConfig_InvalidPrimaryColor(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.PrimaryColor = "not-a-color"
	assertValidationError(t, cfg, "PrimaryColor")
}

func TestValidateConfig_ValidPrimaryColor(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.PrimaryColor = "#4f46e5"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_ValidShortHexColor(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.PrimaryColor = "#f00"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_InvalidSecondaryColor(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.SecondaryColor = "rgb(0,0,0)"
	assertValidationError(t, cfg, "SecondaryColor")
}

func TestValidateConfig_InvalidSidebarColor(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.SidebarColor = "#GGHHII"
	assertValidationError(t, cfg, "SidebarColor")
}

func TestValidateConfig_InvalidSidebarTextColor(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.SidebarTextColor = "white"
	assertValidationError(t, cfg, "SidebarTextColor")
}

func TestValidateConfig_InvalidBorderRadius(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.BorderRadius = "abc"
	assertValidationError(t, cfg, "BorderRadius")
}

func TestValidateConfig_ValidBorderRadius(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.BorderRadius = "0.5rem"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_ZeroBorderRadius(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.BorderRadius = "0"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_EmptyBrandingIsValid(t *testing.T) {
	cfg := validTestConfig()
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error with empty branding, got: %v", err)
	}
}

func TestValidateConfig_EmptyCustomCSSIsValid(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.CustomCSS = ""
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error for empty CustomCSS, got: %v", err)
	}
}

func TestValidateConfig_ValidCustomCSSWithDarkThemeSelector(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.CustomCSS = `[data-bs-theme="dark"] { --bs-body-bg: #121212; }`
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error for short CustomCSS, got: %v", err)
	}
}

func TestValidateConfig_CustomCSSTooLong(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.CustomCSS = strings.Repeat("a", 65537)
	assertValidationError(t, cfg, "CustomCSS")
	assertValidationError(t, cfg, "65536")
}

func TestValidateConfig_CustomCSSRejectsStyleClose(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.CustomCSS = "body{}</Style>"
	assertValidationError(t, cfg, "CustomCSS")
}

func TestValidateConfig_CustomCSSRejectsScript(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.CustomCSS = "body{}<SCRIPT src=x>"
	assertValidationError(t, cfg, "CustomCSS")
}

func TestValidateConfig_CustomCSSRejectsJavascriptURI(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.CustomCSS = "a { background: url(JavaScript:alert(1)); }"
	assertValidationError(t, cfg, "CustomCSS")
}

func TestValidateConfig_EmptyFaviconURLIsValid(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.FaviconURL = ""
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error for empty FaviconURL, got: %v", err)
	}
}

func TestValidateConfig_HTTPSFaviconURLIsValid(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.FaviconURL = "https://example.com/favicon.ico"
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error for https FaviconURL, got: %v", err)
	}
}

func TestValidateConfig_MissingFaviconFileRejected(t *testing.T) {
	cfg := validTestConfig()
	cfg.Admin.Branding.FaviconURL = "/no/such/favicon.ico"
	assertValidationError(t, cfg, "FaviconURL")
}

func TestValidateConfig_FaviconFileTooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "favicon.ico")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxBrandingFileSize + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := validTestConfig()
	cfg.Admin.Branding.FaviconURL = path
	assertValidationError(t, cfg, "FaviconURL")
}

func validTestConfig() Config {
	cfg := DefaultConfig()
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.DBName = "testdb"
	cfg.Database.User = "postgres"
	cfg.JWT.Secret = "a-test-secret-that-is-at-least-32-characters-long"
	return cfg
}

func assertValidationError(t *testing.T, cfg Config, want string) {
	t.Helper()
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}
