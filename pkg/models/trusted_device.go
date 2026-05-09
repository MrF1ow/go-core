package models

import (
	"time"

	"github.com/google/uuid"
)

// TrustedDevice represents a device that a user has opted to trust for 30 days,
// allowing 2FA to be skipped on subsequent logins from that device.
// Scoped per app + user to support multi-tenancy.
type TrustedDevice struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	AppID      uuid.UUID `json:"app_id"`
	TokenHash  string    `json:"-"`    // SHA-256 hex of the plaintext device token (never stored in plain)
	Name       string    `json:"name"` // Human-readable label auto-generated from User-Agent
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"` // IPv4 or IPv6
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName overrides the default table name
func (TrustedDevice) TableName() string {
	return "trusted_devices"
}
