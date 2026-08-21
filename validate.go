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

const maxBrandingFileSize = 1 << 20 // 1 MiB

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
	if p := cfg.Admin.AdminBasePath; p != "" {
		if !strings.HasPrefix(p, "/") {
			return fmt.Errorf("core: Config.Admin.AdminBasePath must start with /")
		}
		if strings.HasSuffix(p, "/") {
			return fmt.Errorf("core: Config.Admin.AdminBasePath must not end with /")
		}
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
	if b.CustomCSS != "" {
		if len(b.CustomCSS) > 65536 {
			return fmt.Errorf("core: Config.Admin.Branding.CustomCSS exceeds 65536 bytes")
		}
		lower := strings.ToLower(b.CustomCSS)
		for _, needle := range []string{"</style", "<script", "javascript:"} {
			if strings.Contains(lower, needle) {
				return fmt.Errorf("core: Config.Admin.Branding.CustomCSS must not contain %s", needle)
			}
		}
	}
	if b.LogoURL != "" && !isRemoteAssetURL(b.LogoURL) {
		if err := validateBrandingFile("LogoURL", b.LogoURL); err != nil {
			return err
		}
	}
	if b.FaviconURL != "" && !isRemoteAssetURL(b.FaviconURL) {
		if err := validateBrandingFile("FaviconURL", b.FaviconURL); err != nil {
			return err
		}
	}
	return nil
}

func isRemoteAssetURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func validateBrandingFile(field, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("core: Config.Admin.Branding.%s file not found: %s", field, path)
	}
	if info.Size() > maxBrandingFileSize {
		return fmt.Errorf("core: Config.Admin.Branding.%s file too large (%d bytes, max %d)", field, info.Size(), maxBrandingFileSize)
	}
	return nil
}
