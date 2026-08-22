package operator

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEnvKeyRosterEntry_JSONOmitsDisabled(t *testing.T) {
	if EnvKeyRosterEntry().Disabled != nil {
		t.Fatal("env key must omit disabled")
	}
	raw, err := json.Marshal(EnvKeyRosterEntry())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "disabled") {
		t.Fatalf("env key JSON included disabled: %s", raw)
	}
}

func TestBuildRoster_PrependsEnvThenKeysThenAccounts(t *testing.T) {
	keyID := uuid.New()
	accountID := uuid.New()
	keys := []RosterEntry{{Kind: string(KindAPIKey), DisplayName: "ops", RoleName: RoleAdmin, KeyID: &keyID}}
	accounts := []RosterEntry{{Kind: string(KindGUIAccount), DisplayName: "ada", RoleName: RoleViewer, AccountID: &accountID}}
	got := BuildRoster(EnvKeyRosterEntry(), keys, accounts)
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Kind != string(KindEnvKey) || got[0].RoleName != RoleSuperadmin || got[0].KeyID != nil {
		t.Fatalf("env row = %+v", got[0])
	}
	if got[1].KeyID == nil || *got[1].KeyID != keyID {
		t.Fatalf("key row = %+v", got[1])
	}
	if got[2].AccountID == nil || *got[2].AccountID != accountID {
		t.Fatalf("account row = %+v", got[2])
	}
}

func TestBuildRoster_TruncatesToExportMaxRows(t *testing.T) {
	keys := make([]RosterEntry, ExportMaxRows)
	got := BuildRoster(EnvKeyRosterEntry(), keys, nil)
	if len(got) != ExportMaxRows {
		t.Fatalf("len = %d, want %d", len(got), ExportMaxRows)
	}
	if got[0].Kind != string(KindEnvKey) {
		t.Fatalf("first kind = %q", got[0].Kind)
	}
}

func TestBuildRoster_KeepsExpiredKeys(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	keys := []RosterEntry{{Kind: string(KindAPIKey), DisplayName: "old", ExpiresAt: &past, Revoked: false}}
	got := BuildRoster(EnvKeyRosterEntry(), keys, nil)
	if len(got) != 2 || got[1].ExpiresAt == nil {
		t.Fatalf("expired key dropped: %+v", got)
	}
}

func TestRoleNameForID_FrozenRoles(t *testing.T) {
	name, ok := RoleNameForID(RoleIDSupport)
	if !ok || name != RoleSupport {
		t.Fatalf("got %q %v", name, ok)
	}
	if _, ok := RoleNameForID(uuid.Nil); ok {
		t.Fatal("nil id mapped")
	}
}
