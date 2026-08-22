package operator

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

// SystemRole is one frozen seeded operator job.
type SystemRole struct {
	ID   uuid.UUID
	Name string
}

// ErrIAMAssignmentDenied is returned when a principal without admin_iam:write
// tries to stamp a non-viewer role on an admin API key.
var ErrIAMAssignmentDenied = errors.New("operator IAM assignment denied")

var errUnknownOperatorRole = errors.New("unknown operator role")

const adminAPIKeyType = "admin"

func systemRoles() []SystemRole {
	return []SystemRole{
		{ID: RoleIDViewer, Name: RoleViewer},
		{ID: RoleIDSupport, Name: RoleSupport},
		{ID: RoleIDAdmin, Name: RoleAdmin},
		{ID: RoleIDSuperadmin, Name: RoleSuperadmin},
	}
}

// AssignableSystemRoles is who the principal may stamp on an admin API key.
// Without admin_iam:write the list is viewer only. With it, all four system jobs.
func AssignableSystemRoles(p Principal) []SystemRole {
	if p.Has(ResAdminIAM, ActionWrite) {
		return systemRoles()
	}
	return []SystemRole{{ID: RoleIDViewer, Name: RoleViewer}}
}

// ParseAssignedAdminRole maps a posted operator_role_id onto an admin API key.
// App keys always return a nil role. Empty posted role stamps viewer and does
// not require IAM. Posted non-viewer without IAM is ErrIAMAssignmentDenied,
// not a silent coerce. Unknown UUIDs stay errors. current is the key's existing
// role on update; nil on create. Posting the same role as current is a no-op
// and does not require IAM.
func ParseAssignedAdminRole(p Principal, postedRoleID, keyType string, current *uuid.UUID) (*uuid.UUID, error) {
	if keyType != adminAPIKeyType {
		return nil, nil
	}
	postedRoleID = strings.TrimSpace(postedRoleID)
	if postedRoleID == "" {
		id := RoleIDViewer
		return &id, nil
	}
	id, err := uuid.Parse(postedRoleID)
	if err != nil {
		return nil, err
	}
	if !IsSystemRoleID(id) {
		return nil, errUnknownOperatorRole
	}
	if current != nil && id == *current {
		return current, nil
	}
	if id != RoleIDViewer && !p.Has(ResAdminIAM, ActionWrite) {
		return nil, ErrIAMAssignmentDenied
	}
	return &id, nil
}

func IsSystemRoleID(id uuid.UUID) bool {
	for _, role := range systemRoles() {
		if role.ID == id {
			return true
		}
	}
	return false
}

func SystemRoleName(id uuid.UUID) string {
	for _, role := range systemRoles() {
		if role.ID == id {
			return role.Name
		}
	}
	return ""
}
