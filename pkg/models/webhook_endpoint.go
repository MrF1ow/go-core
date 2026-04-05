package models

import (
	"time"

	"github.com/google/uuid"
)

// WebhookEndpoint represents a registered webhook URL for a specific application and event type.
// One endpoint per (AppID, EventType) composite — explicit and simple model.
// The signing secret is stored as an HMAC-SHA256 key and shown only once at creation time.
type WebhookEndpoint struct {
	ID        uuid.UUID  `json:"id"`
	AppID     uuid.UUID  `json:"app_id"`
	EventType string     `json:"event_type"`
	URL       string     `json:"url"`
	Secret    string     `json:"-"` // HMAC-SHA256 key, never exposed after creation
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"`
}

// TableName specifies the table name for WebhookEndpoint
func (WebhookEndpoint) TableName() string {
	return "webhook_endpoints"
}

// ValidEventTypes returns the list of supported webhook event types.
var ValidEventTypes = []string{
	"user.registered",
	"user.verified",
	"user.login",
	"user.password_changed",
	"2fa.enabled",
	"2fa.disabled",
	"user.deleted",
	"social.linked",
	"social.unlinked",
}
