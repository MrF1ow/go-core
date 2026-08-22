package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AdminAccount represents a system-level admin user for the Admin GUI.
// These are separate from regular User accounts and are not scoped to any application.
type AdminAccount struct {
	ID             uuid.UUID  `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	PasswordHash   string     `json:"-"`
	OperatorRoleID uuid.UUID  `json:"operator_role_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`

	// Two-Factor Authentication fields
	TwoFAEnabled       bool            `json:"two_fa_enabled"`
	TwoFAMethod        string          `json:"two_fa_method"`
	TwoFASecret        string          `json:"-"`
	TwoFARecoveryCodes json.RawMessage `json:"-"`

	// Magic Link authentication
	MagicLinkEnabled bool `json:"magic_link_enabled"`

	// Backup email for 2FA recovery (separate from primary admin email)
	BackupEmail         string `json:"backup_email,omitempty"`
	BackupEmailVerified bool   `json:"backup_email_verified"`
}

// TableName overrides the default table name
func (AdminAccount) TableName() string {
	return "admin_accounts"
}
