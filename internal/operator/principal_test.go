package operator

import (
	"testing"

	"github.com/google/uuid"
)

func TestPrincipal_HasExactGrantOnly(t *testing.T) {
	p := NewPrincipal(KindAPIKey, RoleViewer, GrantsFor(RoleViewer))
	if !p.Has(ResUsers, ActionRead) {
		t.Fatal("viewer should have users:read")
	}
	if p.Has(ResUsers, ActionWrite) {
		t.Fatal("viewer must not have users:write")
	}
	if p.Has(ResTenants, ActionRead) {
		t.Fatal("viewer must not have tenants:read")
	}
}

func TestPrincipal_NoWildcard(t *testing.T) {
	p := NewPrincipal(KindAPIKey, RoleViewer, []string{"users:*", "*"})
	if p.Has(ResUsers, ActionRead) {
		t.Fatal("wildcards are not grants in v1")
	}
}

func TestSuperadminPrincipal_HasAdminIAM(t *testing.T) {
	p := SuperadminPrincipal(KindEnvKey)
	if p.Kind != KindEnvKey {
		t.Fatalf("kind = %q", p.Kind)
	}
	if p.RoleName != RoleSuperadmin {
		t.Fatalf("role = %q", p.RoleName)
	}
	if !p.Has(ResAdminIAM, ActionWrite) {
		t.Fatal("env superadmin must have admin_iam:write")
	}
}

func TestPrincipal_EmptyPermsDeny(t *testing.T) {
	p := NewPrincipal(KindAPIKey, RoleViewer, nil)
	if p.Has(ResDashboard, ActionRead) {
		t.Fatal("empty grant must deny")
	}
}

func TestPrincipal_KeyIDOptional(t *testing.T) {
	id := uuid.New()
	p := SuperadminPrincipal(KindAPIKey)
	p.KeyID = &id
	if p.KeyID == nil || *p.KeyID != id {
		t.Fatal("key id should round-trip")
	}
}

func TestPrincipal_AppIDOptional(t *testing.T) {
	p := SuperadminPrincipal(KindGUIAccount)
	if p.AppID != nil {
		t.Fatal("app id should be nil by default")
	}
	id := uuid.New()
	p.AppID = &id
	if p.AppID == nil || *p.AppID != id {
		t.Fatal("app id should round-trip")
	}
}
