package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EmailType represents a category of email that the system can send.
// System-defined types (is_system=true) cannot be deleted.
// The Variables field stores the list of available template variables as JSONB.
type EmailType struct {
	ID             uuid.UUID       `json:"id"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	DefaultSubject string          `json:"default_subject"`
	Variables      json.RawMessage `json:"variables"` // [{name, description, required}]
	IsSystem       bool            `json:"is_system"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// TableName specifies the table name for EmailType.
func (EmailType) TableName() string {
	return "email_types"
}

// Variable source constants indicate where a variable's value is automatically resolved from.
const (
	VarSourceUser     = "user"     // Auto-resolved from user profile fields
	VarSourceSetting  = "setting"  // Auto-resolved from app/system settings
	VarSourceExplicit = "explicit" // Must be passed explicitly by the caller
)

// EmailTypeVariable describes a single template variable available for an email type.
// Source indicates where the value is auto-resolved from ("user", "setting", "explicit", or empty for any).
// DefaultValue is a static fallback used when no other source provides a value.
type EmailTypeVariable struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value,omitempty"`
	Source       string `json:"source,omitempty"`
}
