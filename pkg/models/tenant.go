package models

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a customer or organization that owns applications
type Tenant struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Apps      []Application `json:"apps"`
}
