package models

import (
	"time"

	"github.com/google/uuid"
)

// OIDCAuthCode represents a single-use authorization code issued during the
// OAuth2 Authorization Code flow. The code is exchanged at the token endpoint
// for an access token + ID token within a short expiry window.
type OIDCAuthCode struct {
	ID    uuid.UUID `json:"id"`
	AppID uuid.UUID `json:"app_id"`

	// ClientID of the OIDC client that initiated the authorization request
	ClientID string `json:"client_id"`

	// UserID of the end user who approved the authorization request
	UserID uuid.UUID `json:"user_id"`

	// Code is the random single-use authorization code sent to the redirect_uri
	Code string `json:"code"` // #nosec G101 -- random code, not a secret credential

	// RedirectURI must exactly match the redirect_uri used in the token request
	RedirectURI string `json:"redirect_uri"`

	// Scopes granted (space-separated, e.g. "openid profile email")
	Scopes string `json:"scopes"`

	// Nonce from the original authorization request — echoed into the ID token
	Nonce string `json:"nonce"`

	// PKCE fields
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"` // "S256"

	// ExpiresAt: codes expire after a short window (default 10 minutes)
	ExpiresAt time.Time `json:"expires_at"`

	// Used: true after the code has been exchanged (prevents replay attacks)
	Used bool `json:"used"`

	CreatedAt time.Time `json:"created_at"`
}
