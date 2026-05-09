package models

import (
	"time"

	"github.com/google/uuid"
)

// WebhookDelivery represents a single delivery attempt for a webhook event.
// Full response details are stored for debugging and retry purposes.
type WebhookDelivery struct {
	ID           uuid.UUID  `json:"id"`
	EndpointID   uuid.UUID  `json:"endpoint_id"`
	AppID        uuid.UUID  `json:"app_id"`
	EventType    string     `json:"event_type"`
	Payload      string     `json:"payload"`       // JSON payload sent
	Attempt      int        `json:"attempt"`       // Attempt number (1-based)
	StatusCode   int        `json:"status_code"`   // HTTP response code (0 = no response)
	ResponseBody string     `json:"response_body"` // First 1KB of response
	LatencyMs    int64      `json:"latency_ms"`    // Round-trip time in milliseconds
	Success      bool       `json:"success"`       // true = 2xx response
	ErrorMessage string     `json:"error_message"` // Network or timeout error
	NextRetryAt  *time.Time `json:"next_retry_at"` // nil = no more retries
	CreatedAt    time.Time  `json:"created_at"`

	// Relationship (for preloading, not always needed)
	Endpoint *WebhookEndpoint `json:"endpoint,omitempty"`
}

// TableName specifies the table name for WebhookDelivery
func (WebhookDelivery) TableName() string {
	return "webhook_deliveries"
}
