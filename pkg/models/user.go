package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents the core user entity in our system
type User struct {
	ID                 uuid.UUID       `json:"id"`
	AppID              uuid.UUID       `json:"app_id"`
	Email              string          `json:"email"`
	PasswordHash       string          `json:"-"`                              // Stored hashed, not exposed via JSON - not required for social logins
	EmailVerified      bool            `json:"email_verified"`
	IsActive           bool            `json:"is_active"`
	Name               string          `json:"name"`                           // Full name from social login or user input
	FirstName          string          `json:"first_name"`                     // First name from social login
	LastName           string          `json:"last_name"`                      // Last name from social login
	ProfilePicture     string          `json:"profile_picture"`                // Profile picture URL from social login
	Locale             string          `json:"locale"`                         // User's locale/language preference
	TwoFAEnabled       bool            `json:"two_fa_enabled"`
	TwoFAMethod        string          `json:"two_fa_method"`                  // User's chosen 2FA method: "totp" or "email"
	TwoFASecret        string          `json:"-"`                              // Stored encrypted, not exposed via JSON
	TwoFARecoveryCodes json.RawMessage `json:"-"`                              // Stored encrypted, not exposed via JSON
	// Backup email for 2FA recovery (separate from login email)
	BackupEmail         string `json:"backup_email,omitempty"`
	BackupEmailVerified bool   `json:"backup_email_verified"`
	// Previous 2FA method/secret saved when switching to backup_email 2FA so it can be restored on disable
	TwoFAPreviousMethod string `json:"-"`
	TwoFAPreviousSecret string `json:"-"`
	// Phone number for SMS-based recovery
	PhoneNumber   string     `json:"phone_number,omitempty"`
	PhoneVerified bool       `json:"phone_verified"`
	LockedAt      *time.Time `json:"locked_at,omitempty"`      // When the account was locked (nil = not locked)
	LockReason    string     `json:"lock_reason,omitempty"`    // Reason for lockout (e.g., "Too many failed login attempts")
	LockExpiresAt *time.Time `json:"lock_expires_at,omitempty"` // When the lockout expires (nil = permanent until admin unlock)
	// Password history and expiry tracking
	PasswordHistory   json.RawMessage `json:"-"`                              // Array of previous bcrypt hashes (for history enforcement)
	PasswordChangedAt *time.Time      `json:"password_changed_at,omitempty"`  // When the password was last changed (nil = never changed)
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	SocialAccounts    []SocialAccount `json:"social_accounts"`                // One-to-many relationship
}
