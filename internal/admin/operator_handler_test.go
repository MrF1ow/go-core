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
	if len(row) != 11 || row[8] != "" || row[9] != "" || row[10] != "" {
		t.Fatalf("env disabled column = %#v", row)
	}
}

func TestAccountRosterEntry_DisabledFromDisabledAt(t *testing.T) {
	now := time.Now()
	appID := uuid.New()
	enabled := accountRosterEntry(models.AdminAccount{
		ID:       uuid.New(),
		Username: "ada",
		AppID:    &appID,
	}, operator.RoleViewer)
	if enabled.Disabled == nil || *enabled.Disabled {
		t.Fatalf("enabled = %+v", enabled)
	}
	if enabled.AppID == nil || *enabled.AppID != appID {
		t.Fatalf("app ID = %v, want %s", enabled.AppID, appID)
	}
	disabled := accountRosterEntry(models.AdminAccount{
		ID:         uuid.New(),
		Username:   "bob",
		DisabledAt: &now,
	}, operator.RoleViewer)
	if disabled.Disabled == nil || !*disabled.Disabled {
		t.Fatalf("disabled = %+v", disabled)
	}
	if disabled.AppID != nil {
		t.Fatalf("unbound app ID = %v", disabled.AppID)
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
	if row[9] != "" || row[10] != "" {
		t.Fatalf("unbound app columns = %#v", row)
	}
}

func TestRosterCSVRow_BoundAccountApp(t *testing.T) {
	appID := uuid.New()
	id := uuid.New()
	row := rosterCSVRow(operator.RosterEntry{
		Kind:        string(operator.KindGUIAccount),
		DisplayName: "ada",
		RoleName:    operator.RoleViewer,
		AccountID:   &id,
		AppID:       &appID,
		AppName:     "Acme",
	})
	if row[9] != appID.String() || row[10] != "Acme" {
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

func TestUpdateOperatorAccountAppID_Override(t *testing.T) {
	called := false
	id := uuid.New()
	appID := uuid.New()
	h := &Handler{
		UpdateAccountAppID: func(gotID uuid.UUID, gotApp *uuid.UUID) error {
			called = true
			if gotID != id {
				t.Fatalf("id = %s, want %s", gotID, id)
			}
			if gotApp == nil || *gotApp != appID {
				t.Fatalf("app = %v, want %s", gotApp, appID)
			}
			return nil
		},
	}
	if err := h.updateOperatorAccountAppID(id, &appID); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("override not used")
	}
}
