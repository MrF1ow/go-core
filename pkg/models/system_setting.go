package models

import (
	"time"
)

// SystemSetting represents a configurable system setting stored in the database.
// Settings follow a resolution priority: env var (if set) > DB value > hardcoded default.
// The Key field is the primary key and matches the environment variable name (e.g., "ACCESS_TOKEN_EXPIRATION_MINUTES").
type SystemSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Category  string    `json:"category"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for SystemSetting.
func (SystemSetting) TableName() string {
	return "system_settings"
}
