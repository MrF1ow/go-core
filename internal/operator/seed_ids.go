package operator

import "github.com/google/uuid"

// Stable IDs used by migrations/016_operator_rbac.sql.
var (
	RoleIDSuperadmin = uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	RoleIDAdmin      = uuid.MustParse("d0000000-0000-0000-0000-000000000002")
	RoleIDSupport    = uuid.MustParse("d0000000-0000-0000-0000-000000000003")
	RoleIDViewer     = uuid.MustParse("d0000000-0000-0000-0000-000000000004")
)
