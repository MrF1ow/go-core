package webauthn

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var cfg struct {
	RPID        string
	RPName      string
	RPOrigins   []string
	FrontendURL string
	AppName     string
	AdminURL    string
}

// Init configures the webauthn package with the required relying party settings.
func Init(rpID, rpName string, rpOrigins []string, frontendURL, appName, adminURL string) {
	cfg.RPID = rpID
	cfg.RPName = rpName
	cfg.RPOrigins = rpOrigins
	cfg.FrontendURL = frontendURL
	cfg.AppName = appName
	cfg.AdminURL = adminURL
}

// GetWebAuthn creates a configured webauthn.WebAuthn instance for the given application.
// It resolves configuration from application settings and falls back to environment variables.
func GetWebAuthn(app *models.Application) (*webauthn.WebAuthn, error) {
	rpID := resolveRPID(app)
	rpName := resolveRPName(app)
	rpOrigins := resolveRPOrigins(app)

	if rpID == "" {
		return nil, fmt.Errorf("WebAuthn RP ID is not configured. Set WEBAUTHN_RP_ID or FRONTEND_URL")
	}

	waCfg := &webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         rpName,
		RPOrigins:             rpOrigins,
		AttestationPreference: protocol.PreferNoAttestation, // Most common for passkeys
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			// For passkeys (discoverable credentials / resident keys)
			ResidentKey:             protocol.ResidentKeyRequirementPreferred,
			UserVerification:        protocol.VerificationPreferred,
			AuthenticatorAttachment: "",
		},
	}

	return webauthn.New(waCfg)
}

// GetWebAuthnForPasswordless creates a WebAuthn instance configured specifically
// for passwordless (discoverable credential) login, requiring user verification.
func GetWebAuthnForPasswordless(app *models.Application) (*webauthn.WebAuthn, error) {
	rpID := resolveRPID(app)
	rpName := resolveRPName(app)
	rpOrigins := resolveRPOrigins(app)

	if rpID == "" {
		return nil, fmt.Errorf("WebAuthn RP ID is not configured. Set WEBAUTHN_RP_ID or FRONTEND_URL")
	}

	waCfg := &webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         rpName,
		RPOrigins:             rpOrigins,
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:             protocol.ResidentKeyRequirementRequired, // Required for discoverable credentials
			UserVerification:        protocol.VerificationRequired,           // Must verify identity for passwordless
			AuthenticatorAttachment: "",
		},
	}

	return webauthn.New(waCfg)
}

// resolveRPID determines the Relying Party ID for WebAuthn.
// Priority: WEBAUTHN_RP_ID env var → hostname from app FrontendURL → hostname from FRONTEND_URL.
func resolveRPID(app *models.Application) string {
	// Check configured RP ID first
	rpID := cfg.RPID
	if rpID != "" {
		return rpID
	}

	// Try per-app FrontendURL
	if app != nil && app.FrontendURL != "" {
		if parsed, err := url.Parse(app.FrontendURL); err == nil && parsed.Hostname() != "" {
			return parsed.Hostname()
		}
	}

	// Fall back to hostname from FrontendURL
	frontendURL := cfg.FrontendURL
	if frontendURL != "" {
		if parsed, err := url.Parse(frontendURL); err == nil && parsed.Hostname() != "" {
			return parsed.Hostname()
		}
	}

	return ""
}

// resolveRPName determines the Relying Party display name.
// Priority: WEBAUTHN_RP_NAME → Application name → APP_NAME → "Auth API".
func resolveRPName(app *models.Application) string {
	rpName := cfg.RPName
	if rpName != "" {
		return rpName
	}

	if app != nil && app.Name != "" {
		return app.Name
	}

	appName := cfg.AppName
	if appName != "" {
		return appName
	}

	return "Auth API"
}

// resolveRPOrigins determines the allowed origins for WebAuthn ceremonies.
// It starts with WEBAUTHN_RP_ORIGINS, then always appends the per-app FrontendURL
// (if set and not already present) so that apps running on different origins than
// the global default are accepted. Falls back to FRONTEND_URL if the list is empty.
func resolveRPOrigins(app *models.Application) []string {
	var result []string

	result = append(result, cfg.RPOrigins...)

	// Always append the per-app FrontendURL when it differs from the global list.
	// This ensures apps hosted on a different origin than the configured RP origins
	// (e.g. a second tenant app on a different port) can still complete passkey ceremonies.
	if app != nil && app.FrontendURL != "" {
		appOrigin := strings.TrimRight(app.FrontendURL, "/")
		found := false
		for _, o := range result {
			if o == appOrigin {
				found = true
				break
			}
		}
		if !found {
			result = append(result, appOrigin)
		}
	}

	if len(result) == 0 {
		if cfg.FrontendURL != "" {
			result = append(result, strings.TrimRight(cfg.FrontendURL, "/"))
		}
	}

	return result
}

// GetWebAuthnForAdmin creates a WebAuthn instance for admin GUI passkey ceremonies.
// Admin passkeys are not tied to any application — configuration comes from environment variables.
func GetWebAuthnForAdmin() (*webauthn.WebAuthn, error) {
	rpID := cfg.RPID
	if rpID == "" {
		// Fall back to hostname from AdminURL or FrontendURL
		for _, u := range []string{cfg.AdminURL, cfg.FrontendURL} {
			if u != "" {
				if parsed, err := url.Parse(u); err == nil && parsed.Hostname() != "" {
					rpID = parsed.Hostname()
					break
				}
			}
		}
	}
	if rpID == "" {
		return nil, fmt.Errorf("WebAuthn RP ID is not configured. Set WEBAUTHN_RP_ID, ADMIN_URL, or FRONTEND_URL")
	}

	rpName := cfg.RPName
	if rpName == "" {
		rpName = cfg.AppName
	}
	if rpName == "" {
		rpName = "Auth API Admin"
	}

	// Origins for admin GUI: start with configured RP origins, then always append
	// AdminURL so admin passkey ceremonies (registered/verified at the admin GUI
	// origin) are accepted even when RP origins are set for tenant apps.
	var rpOrigins []string
	rpOrigins = append(rpOrigins, cfg.RPOrigins...)
	if adminURL := strings.TrimRight(cfg.AdminURL, "/"); adminURL != "" {
		found := false
		for _, o := range rpOrigins {
			if o == adminURL {
				found = true
				break
			}
		}
		if !found {
			rpOrigins = append(rpOrigins, adminURL)
		}
	}
	if len(rpOrigins) == 0 {
		if cfg.FrontendURL != "" {
			rpOrigins = append(rpOrigins, strings.TrimRight(cfg.FrontendURL, "/"))
		}
	}

	waCfg := &webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         rpName,
		RPOrigins:             rpOrigins,
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:             protocol.ResidentKeyRequirementRequired, // Discoverable credentials for passkey login
			UserVerification:        protocol.VerificationPreferred,
			AuthenticatorAttachment: "", // Use platform authenticators only (Windows Hello, Touch ID) to avoid cloud passkey provider issues
		},
	}

	return webauthn.New(waCfg)
}

// GetWebAuthnForAdminLogin creates a WebAuthn instance for admin GUI passkey login ceremonies.
// Uses stricter settings: discoverable credentials required, user verification required.
func GetWebAuthnForAdminLogin() (*webauthn.WebAuthn, error) {
	rpID := cfg.RPID
	if rpID == "" {
		// Fall back to hostname from AdminURL or FrontendURL
		for _, u := range []string{cfg.AdminURL, cfg.FrontendURL} {
			if u != "" {
				if parsed, err := url.Parse(u); err == nil && parsed.Hostname() != "" {
					rpID = parsed.Hostname()
					break
				}
			}
		}
	}
	if rpID == "" {
		return nil, fmt.Errorf("WebAuthn RP ID is not configured. Set WEBAUTHN_RP_ID, ADMIN_URL, or FRONTEND_URL")
	}

	rpName := cfg.RPName
	if rpName == "" {
		rpName = cfg.AppName
	}
	if rpName == "" {
		rpName = "Auth API Admin"
	}

	// Origins for admin GUI: start with configured RP origins, then always append
	// AdminURL so admin passkey ceremonies (registered/verified at the admin GUI
	// origin) are accepted even when RP origins are set for tenant apps.
	var rpOrigins []string
	rpOrigins = append(rpOrigins, cfg.RPOrigins...)
	if adminURL := strings.TrimRight(cfg.AdminURL, "/"); adminURL != "" {
		found := false
		for _, o := range rpOrigins {
			if o == adminURL {
				found = true
				break
			}
		}
		if !found {
			rpOrigins = append(rpOrigins, adminURL)
		}
	}
	if len(rpOrigins) == 0 {
		if cfg.FrontendURL != "" {
			rpOrigins = append(rpOrigins, strings.TrimRight(cfg.FrontendURL, "/"))
		}
	}

	waCfg := &webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         rpName,
		RPOrigins:             rpOrigins,
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:             protocol.ResidentKeyRequirementRequired, // Discoverable credentials required for login
			UserVerification:        protocol.VerificationRequired,           // Must verify identity for passwordless login
			AuthenticatorAttachment: "",                                      // Use platform authenticators only, consistent with admin registration
		},
	}

	return webauthn.New(waCfg)
}
