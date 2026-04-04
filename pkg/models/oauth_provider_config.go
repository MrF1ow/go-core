package models

import (
	"time"

	"github.com/google/uuid"
)

// OAuthProviderConfig stores OAuth credentials for a specific application and provider
type OAuthProviderConfig struct {
	ID           uuid.UUID `json:"id"`
	AppID        uuid.UUID `json:"app_id"`
	Provider     string    `json:"provider"` // google, facebook, github
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"-"` // Stored encrypted, not exposed via JSON
	RedirectURL  string    `json:"redirect_url"`
	IsEnabled    bool      `json:"is_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName overrides the default table name
func (OAuthProviderConfig) TableName() string {
	return "oauth_provider_configs"
}
