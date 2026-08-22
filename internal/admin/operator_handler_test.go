package admin

import (
	"testing"
	"time"

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
	if len(row) != 9 || row[8] != "" {
		t.Fatalf("env disabled column = %#v", row)
	}
}

func TestAccountRosterEntry_DisabledFromDisabledAt(t *testing.T) {
	now := time.Now()
	enabled := accountRosterEntry(models.AdminAccount{
		ID:       uuid.New(),
		Username: "ada",
	}, operator.RoleViewer)
	if enabled.Disabled == nil || *enabled.Disabled {
		t.Fatalf("enabled = %+v", enabled)
	}
	disabled := accountRosterEntry(models.AdminAccount{
		ID:         uuid.New(),
		Username:   "bob",
		DisabledAt: &now,
	}, operator.RoleViewer)
	if disabled.Disabled == nil || !*disabled.Disabled {
		t.Fatalf("disabled = %+v", disabled)
	}
}

func TestRosterCSVRow_AccountDisabled(t *testing.T) {
	flag := true
	id := uuid.New()
	row := rosterCSVRow(operator.RosterEntry{
		Kind:        string(operator.KindGUIAccount),
		DisplayName: "bob",
		RoleName:    operator.RoleViewer,
		AccountID:   &id,
		Disabled:    &flag,
	})
	if row[8] != "true" {
		t.Fatalf("row = %#v", row)
	}
}

func TestParseOperatorListLimit(t *testing.T) {
	if got := parseOperatorListLimit(""); got != 100 {
		t.Fatalf("empty = %d", got)
	}
	if got := parseOperatorListLimit("0"); got != 100 {
		t.Fatalf("zero = %d", got)
	}
	if got := parseOperatorListLimit("nope"); got != 100 {
		t.Fatalf("junk = %d", got)
	}
	if got := parseOperatorListLimit("250"); got != 250 {
		t.Fatalf("in range = %d", got)
	}
	if got := parseOperatorListLimit("1000"); got != 1000 {
		t.Fatalf("max = %d", got)
	}
	if got := parseOperatorListLimit("1001"); got != 1000 {
		t.Fatalf("over max = %d", got)
	}
	if got := parseOperatorListLimit("2147483648"); got != 1000 {
		t.Fatalf("int32 overflow input = %d", got)
	}
}
