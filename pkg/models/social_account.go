package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SocialAccount stores information related to a user's social media logins
type SocialAccount struct {
	ID             uuid.UUID       `json:"id"`
	AppID          uuid.UUID       `json:"app_id"`
	UserID         uuid.UUID       `json:"user_id"`
	Provider       string          `json:"provider"`
	ProviderUserID string          `json:"provider_user_id"` // Composite unique index with Provider and AppID
	Email          string          `json:"email"`            // Email from social provider
	Name           string          `json:"name"`             // Name from social provider
	FirstName      string          `json:"first_name"`       // First name from social provider
	LastName       string          `json:"last_name"`        // Last name from social provider
	ProfilePicture string          `json:"profile_picture"`  // Profile picture URL from social provider
	Username       string          `json:"username"`         // Username/login from social provider (e.g., GitHub login)
	Locale         string          `json:"locale"`           // Locale from social provider
	RawData        json.RawMessage `json:"raw_data"`         // Complete raw JSON data from provider
	AccessToken    string          `json:"-"`                // Stored encrypted, not exposed via JSON
	RefreshToken   string          `json:"-"`                // Stored encrypted, not exposed via JSON
	ExpiresAt      *time.Time      `json:"expires_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
