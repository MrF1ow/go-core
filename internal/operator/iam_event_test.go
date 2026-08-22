package operator

import "testing"

func TestWouldLeaveLastSuperadmin(t *testing.T) {
	tests := []struct {
		count  int
		target bool
		want   bool
	}{
		{0, true, false},
		{1, true, true},
		{1, false, false},
		{2, true, false},
		{2, false, false},
	}
	for _, tt := range tests {
		got := WouldLeaveLastSuperadmin(tt.count, tt.target)
		if got != tt.want {
			t.Fatalf("WouldLeaveLastSuperadmin(%d, %v) = %v, want %v", tt.count, tt.target, got, tt.want)
		}
	}
}

func TestNewSetupCLICreateEvent(t *testing.T) {
	accountID := RoleIDViewer
	roleID := RoleIDSuperadmin
	ev := NewSetupCLICreateEvent(accountID, roleID)
	if ev.ActorKind != ActorKindSetupCLI {
		t.Fatalf("actor_kind = %q, want %q", ev.ActorKind, ActorKindSetupCLI)
	}
	if ev.TargetKind != KindGUIAccount {
		t.Fatalf("target_kind = %q", ev.TargetKind)
	}
	if ev.TargetAccountID == nil || *ev.TargetAccountID != accountID {
		t.Fatalf("target_account_id = %v", ev.TargetAccountID)
	}
	if ev.NewRoleID == nil || *ev.NewRoleID != roleID {
		t.Fatalf("new_role_id = %v", ev.NewRoleID)
	}
	if ev.Action != ActionCreatePrincipal {
		t.Fatalf("action = %q", ev.Action)
	}
	if ev.ActorKeyID != nil || ev.ActorAccountID != nil {
		t.Fatalf("setup_cli must not set request actor ids")
	}
}
