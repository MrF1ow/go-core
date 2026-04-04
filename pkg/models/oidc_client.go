package models

import (
	"time"

	"github.com/google/uuid"
)

// OIDCClient represents a registered OIDC/OAuth2 relying party (client application)
// that delegates authentication to this OIDC provider.
// Each client is scoped to an Application (AppID).
type OIDCClient struct {
	ID    uuid.UUID `json:"id"`
	AppID uuid.UUID `json:"app_id"`

	// Human-readable name shown on the consent screen
	Name        string `json:"name"`
	Description string `json:"description"`

	// OIDC client credentials
	// ClientID is a public identifier — safe to expose to end users
	ClientID string `json:"client_id"`
	// ClientSecretHash is the bcrypt hash of the client secret — never exposed via JSON
	ClientSecretHash string `json:"-"`

	// JSON array of allowed redirect URIs
	// Example: '["https://app.example.com/callback"]'
	RedirectURIs string `json:"redirect_uris"`

	// Comma-separated list of allowed grant types
	// Supported: "authorization_code", "client_credentials", "refresh_token"
	AllowedGrantTypes string `json:"allowed_grant_types"`

	// Comma-separated list of allowed OIDC scopes
	// Supported: "openid", "profile", "email", "roles"
	AllowedScopes string `json:"allowed_scopes"`

	// RequireConsent: if true, shows consent screen; if false, auto-approves all scopes
	RequireConsent bool `json:"require_consent"`

	// IsConfidential: true = confidential client (has secret), false = public client (PKCE only)
	IsConfidential bool `json:"is_confidential"`

	// PKCERequired: if true, PKCE code_challenge is mandatory (even for confidential clients)
	PKCERequired bool `json:"pkce_required"`

	// LogoURL: optional URL to client logo shown on consent screen
	LogoURL string `json:"logo_url"`

	// LoginTheme controls the color scheme of OIDC login/consent pages for this client.
	// "auto" (default) follows the user's OS preference; "light" and "dark" force a mode.
	LoginTheme string `json:"login_theme"`

	// LoginPrimaryColor is an optional hex color (e.g. "#4f46e5") that overrides Bootstrap's
	// default primary blue on OIDC pages. Empty string means use Bootstrap default (#0d6efd).
	LoginPrimaryColor string `json:"login_primary_color"`

	// IsActive: soft-disable a client without deleting it
	IsActive bool `json:"is_active"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
