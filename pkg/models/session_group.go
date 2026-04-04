package models

import (
	"time"

	"github.com/google/uuid"
)

// SessionGroup is a named group of applications that share authentication state.
// When a user is authenticated in any app in the group they can obtain tokens
// for any other app in the group via the SSO exchange flow without re-entering
// credentials (similar to Google's cross-product SSO).
type SessionGroup struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	// GlobalLogout controls whether logging out of one app in the group
	// revokes the user's sessions in all other apps of the group.
	GlobalLogout bool              `json:"global_logout"`
	Apps         []SessionGroupApp `json:"apps,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// SessionGroupApp is the join table linking an Application to a SessionGroup.
// An application can belong to at most one session group (enforced by the
// unique index on AppID).
type SessionGroupApp struct {
	SessionGroupID uuid.UUID   `json:"session_group_id"`
	AppID          uuid.UUID   `json:"app_id"`
	App            Application `json:"app,omitempty"`
	AddedAt        time.Time   `json:"added_at"`
}
