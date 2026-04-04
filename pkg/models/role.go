package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a named role scoped to an application
type Role struct {
	ID          uuid.UUID    `json:"id"`
	AppID       uuid.UUID    `json:"app_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"` // System roles cannot be deleted
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// Permission represents a granular permission (resource:action)
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserRole represents the assignment of a role to a user within an application
type UserRole struct {
	UserID     uuid.UUID  `json:"user_id"`
	RoleID     uuid.UUID  `json:"role_id"`
	AppID      uuid.UUID  `json:"app_id"` // Denormalized from Role for fast lookup
	AssignedAt time.Time  `json:"assigned_at"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty"` // Who assigned this role (nullable for system assignments)
	Role       Role       `json:"role,omitempty"`
	User       User       `json:"user,omitempty"`
}
