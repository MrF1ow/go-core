package operator

import "github.com/google/uuid"

// RestrictAppQuery returns the bound app when set, otherwise the requested value.
func RestrictAppQuery(bound *uuid.UUID, requested string) string {
	if bound != nil {
		return bound.String()
	}
	return requested
}

// ForeignApp is true when bound is set and the resource belongs to another app.
func ForeignApp(bound *uuid.UUID, resource uuid.UUID) bool {
	if bound == nil {
		return false
	}
	return *bound != resource
}

// ForeignAppID is true when bound is set and raw is missing, invalid, or another app.
func ForeignAppID(bound *uuid.UUID, raw string) bool {
	id, err := uuid.Parse(raw)
	if err != nil {
		return bound != nil
	}
	return ForeignApp(bound, id)
}
