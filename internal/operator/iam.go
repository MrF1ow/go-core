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

var (
	// ErrIAMAssignmentDenied is returned when a principal without admin_iam:write
	// tries to stamp a non-viewer role on an admin API key.
	ErrIAMAssignmentDenied = errors.New("operator IAM assignment denied")
	// ErrAdminIAMOnCustomRole is returned when a custom role grant list includes admin_iam.
	ErrAdminIAMOnCustomRole   = errors.New("custom roles cannot grant admin_iam")
	ErrReservedSystemRoleName = errors.New("system role names cannot be reused")
	ErrSystemRoleImmutable    = errors.New("system operator roles cannot be modified")
	ErrRoleNameTaken          = errors.New("operator role name already exists")
	ErrRoleAssigned           = errors.New("operator role is assigned")
	ErrRoleReferenced         = errors.New("operator role is referenced by IAM events")

	errUnknownOperatorRole  = errors.New("unknown operator role")
	errUnknownOperatorGrant = errors.New("unknown operator grant")
)

// RoleExistsFunc reports whether a role id exists in operator_roles.
type RoleExistsFunc func(id uuid.UUID) (bool, error)

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
// not a silent coerce. System ids skip exists. A non-system id is valid when
// exists reports true; missing or nil lookup is errUnknownOperatorRole.
// current is the key's existing role on update; nil on create. Posting the
// same role as current is a no-op and does not require IAM.
func ParseAssignedAdminRole(p Principal, postedRoleID, keyType string, current *uuid.UUID, exists ...RoleExistsFunc) (*uuid.UUID, error) {
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
		var lookup RoleExistsFunc
		if len(exists) > 0 {
			lookup = exists[0]
		}
		if lookup == nil {
			return nil, errUnknownOperatorRole
		}
		ok, err := lookup(id)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errUnknownOperatorRole
		}
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

// ParseCustomGrants maps posted resource:action keys onto catalog permissions.
// Unknown pairs are errors. admin_iam on a custom role is ErrAdminIAMOnCustomRole.
func ParseCustomGrants(keys []string) ([]Permission, error) {
	out := make([]Permission, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		resource, action, ok := strings.Cut(key, ":")
		if !ok || resource == "" || action == "" {
			return nil, errUnknownOperatorGrant
		}
		if resource == ResAdminIAM {
			return nil, ErrAdminIAMOnCustomRole
		}
		var found *Permission
		for i := range catalog {
			if catalog[i].Resource == resource && catalog[i].Action == action {
				found = &catalog[i]
				break
			}
		}
		if found == nil {
			return nil, errUnknownOperatorGrant
		}
		if _, dup := seen[found.Key()]; dup {
			continue
		}
		seen[found.Key()] = struct{}{}
		out = append(out, *found)
	}
	return out, nil
}

func IsReservedSystemRoleName(name string) bool {
	for _, reserved := range SystemRoleNames() {
		if name == reserved {
			return true
		}
	}
	return false
}
