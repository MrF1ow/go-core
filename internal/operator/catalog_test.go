package operator

import (
	"testing"
)

func TestCatalog_KeysAreUniqueAndComplete(t *testing.T) {
	seen := map[string]struct{}{}
	for _, p := range Catalog() {
		k := p.Key()
		if _, dup := seen[k]; dup {
			t.Fatalf("duplicate permission %q", k)
		}
		seen[k] = struct{}{}
	}
	if len(seen) != 31 {
		t.Fatalf("catalog size = %d, want 31", len(seen))
	}
	for _, need := range []string{
		"dashboard:read",
		"tenants:write",
		"admin_iam:read",
		"admin_iam:write",
		"monitoring:read",
		"logs:read",
	} {
		if _, ok := seen[need]; !ok {
			t.Errorf("missing %q", need)
		}
	}
	if _, ok := seen["dashboard:write"]; ok {
		t.Error("dashboard must be read-only")
	}
	if _, ok := seen["logs:write"]; ok {
		t.Error("logs must be read-only")
	}
	if _, ok := seen["monitoring:write"]; ok {
		t.Error("monitoring must be read-only")
	}
}

func TestGrantsFor_ViewerIsReadOnlySubset(t *testing.T) {
	got := grantSet(GrantsFor(RoleViewer))
	want := grantSet([]string{"dashboard:read", "users:read", "logs:read", "monitoring:read"})
	assertGrantSet(t, got, want)
}

func TestGrantsFor_Support(t *testing.T) {
	got := grantSet(GrantsFor(RoleSupport))
	want := grantSet([]string{
		"dashboard:read",
		"users:read", "users:write",
		"sessions:read", "sessions:write",
		"logs:read",
	})
	assertGrantSet(t, got, want)
}

func TestGrantsFor_AdminExcludesAdminIAM(t *testing.T) {
	got := grantSet(GrantsFor(RoleAdmin))
	if _, ok := got["admin_iam:read"]; ok {
		t.Error("admin must not have admin_iam:read")
	}
	if _, ok := got["admin_iam:write"]; ok {
		t.Error("admin must not have admin_iam:write")
	}
	if _, ok := got["tenants:write"]; !ok {
		t.Error("admin must have tenants:write")
	}
	if len(got) != 29 {
		t.Fatalf("admin grant size = %d, want 29 (catalog minus admin_iam)", len(got))
	}
}

func TestGrantsFor_SuperadminIsFullCatalog(t *testing.T) {
	got := grantSet(GrantsFor(RoleSuperadmin))
	want := grantSet(allGrantKeys())
	assertGrantSet(t, got, want)
	if _, ok := got["admin_iam:write"]; !ok {
		t.Error("superadmin must have admin_iam:write")
	}
}

func TestGrantsFor_UnknownRoleEmpty(t *testing.T) {
	if keys := GrantsFor("not-a-role"); len(keys) != 0 {
		t.Fatalf("unknown role grants = %v, want empty", keys)
	}
}

func TestDefaultRoleForNewPrincipal_IsViewer(t *testing.T) {
	if DefaultRoleForNewPrincipal() != RoleViewer {
		t.Fatalf("default = %q, want %q", DefaultRoleForNewPrincipal(), RoleViewer)
	}
}

func grantSet(keys []string) map[string]struct{} {
	out := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		out[k] = struct{}{}
	}
	return out
}

func assertGrantSet(t *testing.T, got, want map[string]struct{}) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("size got %d want %d\ngot  %v\nwant %v", len(got), len(want), keysOf(got), keysOf(want))
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func keysOf(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
