package models

import (
	"time"

	"github.com/google/uuid"
)

// ApiKeyUsage tracks daily request counts per API key for usage analytics.
// One row is maintained per (api_key_id, period_date) pair using upsert semantics.
type ApiKeyUsage struct {
	ID           uint      `json:"id"`
	ApiKeyID     uuid.UUID `json:"api_key_id"`
	PeriodDate   time.Time `json:"period_date"` // Day bucket (YYYY-MM-DD)
	RequestCount int64     `json:"request_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName specifies the table name for ApiKeyUsage.
func (ApiKeyUsage) TableName() string {
	return "api_key_usages"
}
