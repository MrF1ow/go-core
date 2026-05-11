package core

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	hexColorRegex     = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	borderRadiusRegex = regexp.MustCompile(`^(0|[0-9]+(\.[0-9]+)?(px|rem|em))$`)
)

// ValidateConfig checks that all required Config fields are set.
// Returns a descriptive error for the first missing or invalid field.
// Called by app.New() before any initialization or connections.
func ValidateConfig(cfg Config) error {
	if cfg.Database.Host == "" {
		return fmt.Errorf("core: Config.Database.Host is required")
	}
	if cfg.Database.Port <= 0 {
		return fmt.Errorf("core: Config.Database.Port must be > 0")
	}
	if cfg.Database.DBName == "" {
		return fmt.Errorf("core: Config.Database.DBName is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("core: Config.Database.User is required")
	}
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("core: Config.JWT.Secret is required")
	}
	if len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf("core: Config.JWT.Secret must be at least 32 characters")
	}
	return validateBranding(cfg.Admin.Branding)
}

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
