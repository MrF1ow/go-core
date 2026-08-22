package operator

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestAssignableSystemRoles_WithoutIAMIsViewerOnly(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleAdmin, GrantsFor(RoleAdmin))
	got := AssignableSystemRoles(p)
	if len(got) != 1 || got[0].ID != RoleIDViewer || got[0].Name != RoleViewer {
		t.Fatalf("got %#v", got)
	}
}

func TestAssignableSystemRoles_WithIAMIsFourSystemNames(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleSuperadmin, GrantsFor(RoleSuperadmin))
	got := AssignableSystemRoles(p)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	names := map[string]bool{}
	for _, role := range got {
		names[role.Name] = true
	}
	for _, want := range SystemRoleNames() {
		if !names[want] {
			t.Fatalf("missing %q in %#v", want, got)
		}
	}
}

func TestParseAssignedAdminRole_AdminPostedSuperadminDenied(t *testing.T) {
	p := NewPrincipal(KindAPIKey, RoleAdmin, GrantsFor(RoleAdmin))
	_, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "admin", nil)
	if !errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatalf("err = %v, want ErrIAMAssignmentDenied", err)
	}
}

func TestParseAssignedAdminRole_AdminEmptyPostIsViewer(t *testing.T) {
	p := NewPrincipal(KindAPIKey, RoleAdmin, GrantsFor(RoleAdmin))
	id, err := ParseAssignedAdminRole(p, "", "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != RoleIDViewer {
		t.Fatalf("got %v", id)
	}
}

func TestParseAssignedAdminRole_SuperadminPostedSuperadmin(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleSuperadmin, GrantsFor(RoleSuperadmin))
	id, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != RoleIDSuperadmin {
		t.Fatalf("got %v", id)
	}
}

func TestParseAssignedAdminRole_AppKeyIgnoresPostedSuperadmin(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleAdmin, GrantsFor(RoleAdmin))
	id, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != nil {
		t.Fatalf("got %v, want nil", id)
	}
}

func TestParseAssignedAdminRole_UnknownUUIDRejected(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleSuperadmin, GrantsFor(RoleSuperadmin))
	_, err := ParseAssignedAdminRole(p, uuid.New().String(), "admin", nil)
	if err == nil {
		t.Fatal("unknown role id must fail")
	}
	if errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatal("unknown UUID must not look like an IAM deny")
	}
}

func TestParseAssignedAdminRole_AdminKeepingCurrentSuperadmin(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleAdmin, GrantsFor(RoleAdmin))
	current := RoleIDSuperadmin
	id, err := ParseAssignedAdminRole(p, current.String(), "admin", &current)
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != RoleIDSuperadmin {
		t.Fatalf("got %v", id)
	}
}

func TestParseAssignedAdminRole_AdminChangingSupportToSuperadminDenied(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleAdmin, GrantsFor(RoleAdmin))
	current := RoleIDSupport
	_, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "admin", &current)
	if !errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatalf("err = %v, want ErrIAMAssignmentDenied", err)
	}
}

func TestParseAssignedAdminRole_CustomRoleExistsStamps(t *testing.T) {
	customID := uuid.New()
	p := NewPrincipal(KindGUIAccount, RoleSuperadmin, GrantsFor(RoleSuperadmin))
	exists := func(id uuid.UUID) (bool, error) { return id == customID, nil }
	id, err := ParseAssignedAdminRole(p, customID.String(), "admin", nil, exists)
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != customID {
		t.Fatalf("got %v, want custom id", id)
	}
}

func TestParseAssignedAdminRole_CustomRoleMissingRejected(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleSuperadmin, GrantsFor(RoleSuperadmin))
	exists := func(uuid.UUID) (bool, error) { return false, nil }
	_, err := ParseAssignedAdminRole(p, uuid.New().String(), "admin", nil, exists)
	if err == nil {
		t.Fatal("missing custom role id must fail")
	}
	if errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatal("missing custom role must not look like an IAM deny")
	}
}

func TestParseAssignedAdminRole_ViewerStampCustomDenied(t *testing.T) {
	customID := uuid.New()
	p := NewPrincipal(KindAPIKey, RoleViewer, GrantsFor(RoleViewer))
	exists := func(id uuid.UUID) (bool, error) { return id == customID, nil }
	_, err := ParseAssignedAdminRole(p, customID.String(), "admin", nil, exists)
	if !errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatalf("err = %v, want ErrIAMAssignmentDenied", err)
	}
}

func TestParseCustomGrants_AdminIAMRejected(t *testing.T) {
	_, err := ParseCustomGrants([]string{ResUsers + ":" + ActionRead, ResAdminIAM + ":" + ActionWrite})
	if !errors.Is(err, ErrAdminIAMOnCustomRole) {
		t.Fatalf("err = %v, want ErrAdminIAMOnCustomRole", err)
	}
}

func TestParseCustomGrants_UnknownPairRejected(t *testing.T) {
	_, err := ParseCustomGrants([]string{"not_a_resource:read"})
	if err == nil {
		t.Fatal("unknown catalog pair must fail")
	}
	if errors.Is(err, ErrAdminIAMOnCustomRole) {
		t.Fatal("unknown pair must not look like admin_iam deny")
	}
}

func TestParseCustomGrants_KnownPairsOK(t *testing.T) {
	got, err := ParseCustomGrants([]string{ResUsers + ":" + ActionRead, ResLogs + ":" + ActionRead})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestParseCustomGrants_EmptyOK(t *testing.T) {
	got, err := ParseCustomGrants(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
	got, err = ParseCustomGrants([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestPrincipalHas_ZeroGrantsDenies(t *testing.T) {
	p := NewPrincipal(KindAPIKey, "auditor", nil)
	if p.Has(ResUsers, ActionRead) {
		t.Fatal("zero-grant custom role allowed users:read")
	}
	if p.Has(ResAdminIAM, ActionWrite) {
		t.Fatal("zero-grant custom role allowed admin_iam:write")
	}
}
