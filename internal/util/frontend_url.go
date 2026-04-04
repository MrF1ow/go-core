package util

import (
	"strings"
)

var (
	globalFrontendURL string
	globalAppName     string
)

// Init configures the util package with global defaults.
// Must be called once at application startup.
func Init(frontendURL, appName string) {
	globalFrontendURL = frontendURL
	globalAppName = appName
}

// ResolveFrontendURL returns the effective base frontend URL for an application.
// Resolution priority: per-app FrontendURL > global default > http://localhost:8080
// The returned value always has any trailing slash stripped.
func ResolveFrontendURL(appFrontendURL string) string {
	if u := strings.TrimRight(appFrontendURL, "/"); u != "" {
		return u
	}
	if u := strings.TrimRight(globalFrontendURL, "/"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

// ResolveLinkPath returns the effective path suffix for an email action link.
// If appPath is non-empty it is used as-is; otherwise defaultPath is returned.
// The returned value always has a leading slash and no trailing slash.
func ResolveLinkPath(appPath, defaultPath string) string {
	p := strings.TrimRight(appPath, "/")
	if p == "" {
		p = strings.TrimRight(defaultPath, "/")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// ResolveAppName returns the effective application name for use in emails and UI.
// Resolution priority: global config > "Auth API" default.
func ResolveAppName() string {
	if globalAppName != "" {
		return globalAppName
	}
	return "Auth API"
}

// Default path constants used when per-app overrides are not configured.
const (
	DefaultResetPasswordPath = "/reset-password"
	DefaultMagicLinkPath     = "/magic-link"
	DefaultVerifyEmailPath   = "/verify-email"
)
