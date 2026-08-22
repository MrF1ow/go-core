package admin

import (
	"testing"

	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
)

func TestAccountRoleName_FrozenAndUnknown(t *testing.T) {
	customID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	h := &Handler{}
	if name := h.accountRoleName(models.AdminAccount{OperatorRoleID: operator.RoleIDViewer}); name != operator.RoleViewer {
		t.Fatalf("frozen = %q", name)
	}
	if name := h.accountRoleName(models.AdminAccount{OperatorRoleID: customID}); name != "" {
		t.Fatalf("unknown without lookup = %q", name)
	}
}

func TestRosterCSVRow_EnvHasEmptyID(t *testing.T) {
	row := rosterCSVRow(operator.EnvKeyRosterEntry())
	if row[0] != string(operator.KindEnvKey) || row[1] != "" || row[3] != operator.RoleSuperadmin {
		t.Fatalf("row = %#v", row)
	}
}
