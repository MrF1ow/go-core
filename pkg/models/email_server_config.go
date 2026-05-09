package models

import (
	"time"

	"github.com/google/uuid"
)

// EmailServerConfig stores SMTP server configuration.
// Configs can be scoped to a specific application (AppID set) or global/system-level (AppID nil).
// Multiple configs can exist per application (e.g., transactional, marketing, finance).
// One config per scope is marked as the default (is_default=true).
// Resolution chain for sending emails: app-specific config -> global config -> dev mode (log to stdout).
type EmailServerConfig struct {
	ID           uuid.UUID  `json:"id"`
	AppID        *uuid.UUID `json:"app_id"` // NULL = global/system-level config
	Name         string     `json:"name"`   // Label (e.g., "Transactional", "Marketing")
	SMTPHost     string     `json:"smtp_host"`
	SMTPPort     int        `json:"smtp_port"`
	SMTPUsername string     `json:"smtp_username"`
	SMTPPassword string     `json:"-"` // Not exposed in JSON responses
	FromAddress  string     `json:"from_address"`
	FromName     string     `json:"from_name"`
	UseTLS       bool       `json:"use_tls"`
	IsDefault    bool       `json:"is_default"` // Only one default per scope (app or global)
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName specifies the table name for EmailServerConfig.
func (EmailServerConfig) TableName() string {
	return "email_server_configs"
}
