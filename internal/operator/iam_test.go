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
	_, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "admin")
	if !errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatalf("err = %v, want ErrIAMAssignmentDenied", err)
	}
}

func TestParseAssignedAdminRole_AdminEmptyPostIsViewer(t *testing.T) {
	p := NewPrincipal(KindAPIKey, RoleAdmin, GrantsFor(RoleAdmin))
	id, err := ParseAssignedAdminRole(p, "", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != RoleIDViewer {
		t.Fatalf("got %v", id)
	}
}

func TestParseAssignedAdminRole_SuperadminPostedSuperadmin(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleSuperadmin, GrantsFor(RoleSuperadmin))
	id, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != RoleIDSuperadmin {
		t.Fatalf("got %v", id)
	}
}

func TestParseAssignedAdminRole_AppKeyIgnoresPostedSuperadmin(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleAdmin, GrantsFor(RoleAdmin))
	id, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "app")
	if err != nil {
		t.Fatal(err)
	}
	if id != nil {
		t.Fatalf("got %v, want nil", id)
	}
}

func TestParseAssignedAdminRole_UnknownUUIDRejected(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleSuperadmin, GrantsFor(RoleSuperadmin))
	_, err := ParseAssignedAdminRole(p, uuid.New().String(), "admin")
	if err == nil {
		t.Fatal("unknown role id must fail")
	}
	if errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatal("unknown UUID must not look like an IAM deny")
	}
}

func TestParseAssignedAdminRole_AdminPostedCurrentSuperadminStillDenied(t *testing.T) {
	p := NewPrincipal(KindGUIAccount, RoleAdmin, GrantsFor(RoleAdmin))
	_, err := ParseAssignedAdminRole(p, RoleIDSuperadmin.String(), "admin")
	if !errors.Is(err, ErrIAMAssignmentDenied) {
		t.Fatalf("err = %v, want ErrIAMAssignmentDenied", err)
	}
}
