package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ActivityLog captures essential details about each user action
type ActivityLog struct {
	ID        uuid.UUID       `json:"id"`
	AppID     uuid.UUID       `json:"app_id"`
	UserID    uuid.UUID       `json:"user_id"` // Composite indexes for performance
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	IPAddress string          `json:"ip_address"`
	UserAgent string          `json:"user_agent"`
	Details   json.RawMessage `json:"details"` // Use json.RawMessage for flexible JSONB

	// New fields for smart logging
	Severity  string     `json:"severity"` // CRITICAL, IMPORTANT, INFORMATIONAL
	ExpiresAt *time.Time `json:"expires_at"`                                // Automatic expiration timestamp for cleanup
	IsAnomaly bool       `json:"is_anomaly"`                                    // Flag if this was logged due to anomaly detection
}

// TableName specifies the table name for ActivityLog
func (ActivityLog) TableName() string {
	return "activity_logs"
}
