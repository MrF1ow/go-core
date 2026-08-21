package operator

import (
	"context"

	"github.com/google/uuid"
)

// GrantLookup loads a role's name and grant keys. Implemented by Repository.
type GrantLookup interface {
	RoleGrants(ctx context.Context, roleID uuid.UUID) (roleName string, keys []string, err error)
}
