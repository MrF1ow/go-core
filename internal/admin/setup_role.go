package admin

import (
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
)

func RoleIDForSetupAccount(existingCount int64) uuid.UUID {
	if existingCount == 0 {
		return operator.RoleIDSuperadmin
	}
	return operator.RoleIDViewer
}
