package models

import (
	"time"

	"github.com/google/uuid"
)

// WebAuthnCredential stores a FIDO2/WebAuthn passkey credential.
// Supports both regular users (UserID+AppID) and admin accounts (AdminID).
// For regular users: UserID and AppID are set, AdminID is nil.
// For admin accounts: AdminID is set, UserID and AppID are nil.
type WebAuthnCredential struct {
	ID              uuid.UUID  `json:"id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`  // Regular user (nullable for admin passkeys)
	AppID           *uuid.UUID `json:"app_id,omitempty"`   // Application (nullable for admin passkeys)
	AdminID         *uuid.UUID `json:"admin_id,omitempty"` // Admin account (nullable for user passkeys)
	CredentialID    []byte     `json:"-"`
	PublicKey       []byte     `json:"-"`
	AttestationType string     `json:"attestation_type"`
	AAGUID          []byte     `json:"-"` // Authenticator identifier
	SignCount       uint32     `json:"-"`
	Name            string     `json:"name"`       // User-friendly name ("My MacBook", "YubiKey")
	Transports      string     `json:"transports"` // Comma-separated: "usb,ble,nfc,internal"
	BackupEligible  bool       `json:"backup_eligible"`
	BackupState     bool       `json:"backup_state"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}
