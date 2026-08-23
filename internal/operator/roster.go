package operator

import (
	"time"

	"github.com/google/uuid"
)

// ExportMaxRows is the hard cap for roster list and export, same as activity-log export.
const ExportMaxRows = 10_000

// RosterEntry is one operator principal on the IAM roster.
type RosterEntry struct {
	Kind        string     `json:"kind"`
	DisplayName string     `json:"display_name"`
	RoleName    string     `json:"role"`
	KeyID       *uuid.UUID `json:"key_id,omitempty"`
	AccountID   *uuid.UUID `json:"account_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Revoked     bool       `json:"revoked"`
	Disabled    *bool      `json:"disabled,omitempty"`
	AppID       *uuid.UUID `json:"app_id,omitempty"`
	AppName     string     `json:"app_name,omitempty"`
}

// EnvKeyRosterEntry is the synthetic break-glass row. It is not a table row.
func EnvKeyRosterEntry() RosterEntry {
	return RosterEntry{
		Kind:        string(KindEnvKey),
		DisplayName: string(KindEnvKey),
		RoleName:    RoleSuperadmin,
	}
}

// BuildRoster prepends env, then keys, then accounts, then truncates to ExportMaxRows.
func BuildRoster(env RosterEntry, keys []RosterEntry, accounts []RosterEntry) []RosterEntry {
	n := 1 + len(keys) + len(accounts)
	out := make([]RosterEntry, 0, min(n, ExportMaxRows))
	out = append(out, env)
	out = append(out, keys...)
	out = append(out, accounts...)
	if len(out) > ExportMaxRows {
		return out[:ExportMaxRows]
	}
	return out
}

// RoleNameForID maps a frozen system role id. Unknown ids are false so a caller can look them up.
func RoleNameForID(id uuid.UUID) (string, bool) {
	for _, role := range systemRoles() {
		if role.ID == id {
			return role.Name, true
		}
	}
	return "", false
}
