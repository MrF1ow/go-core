package web

import (
	"strings"
	"testing"

	"github.com/MrF1ow/go-core/internal/operator"
)

func TestTemplateDataCan_NilDenies(t *testing.T) {
	var td TemplateData
	if td.Can("dashboard", "read") {
		t.Fatal("nil can allowed dashboard:read")
	}
}

func TestBuildNav_NilCanYieldsNoGroups(t *testing.T) {
	if groups := buildNav("/gui", nil); len(groups) != 0 {
		t.Fatalf("groups = %#v, want empty", groups)
	}
}

func TestBuildNav_ViewerOmitsTenantsAndEmptyEmail(t *testing.T) {
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleViewer, operator.GrantsFor(operator.RoleViewer))
	groups := buildNav("/gui", p.Has)
	got := flattenNav(groups)

	want := []string{"Dashboard", "Users", "Activity Logs", "System Health"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("labels = %v, want %v", got, want)
	}
	if headingPresent(groups, "Email") {
		t.Fatal("empty Email heading leaked")
	}
	if headingPresent(groups, "Security") {
		t.Fatal("empty Security heading leaked")
	}
	if labelPresent(groups, "Tenants") {
		t.Fatal("viewer nav includes Tenants")
	}
	if labelPresent(groups, "API Keys") {
		t.Fatal("viewer nav includes API Keys")
	}
}

func TestBuildNav_SupportIncludesSessionsOmitsTenants(t *testing.T) {
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleSupport, operator.GrantsFor(operator.RoleSupport))
	groups := buildNav("/gui", p.Has)
	got := flattenNav(groups)
	want := []string{"Dashboard", "Users", "Sessions", "Activity Logs"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("labels = %v, want %v", got, want)
	}
	if labelPresent(groups, "Tenants") {
		t.Fatal("support nav includes Tenants")
	}
}

func TestBuildNav_SuperadminHasEveryRowAndNoAdminIAM(t *testing.T) {
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleSuperadmin, operator.GrantsFor(operator.RoleSuperadmin))
	groups := buildNav("/gui", p.Has)
	got := flattenNav(groups)
	if len(got) != len(navSpec) {
		t.Fatalf("got %d items, want %d spec rows: %v", len(got), len(navSpec), got)
	}
	if labelPresent(groups, "admin_iam") {
		t.Fatal("nav includes admin_iam")
	}
	for _, spec := range navSpec {
		if spec.Resource == "admin_iam" {
			t.Fatal("navSpec contains admin_iam")
		}
	}
}

func TestNavSpecResourcesAreInCatalog(t *testing.T) {
	catalog := map[string]struct{}{}
	for _, p := range operator.Catalog() {
		catalog[p.Resource] = struct{}{}
	}
	for _, spec := range navSpec {
		if _, ok := catalog[spec.Resource]; !ok {
			t.Errorf("nav resource %q not in catalog", spec.Resource)
		}
		if spec.Action != operator.ActionRead {
			t.Errorf("nav %q action %q, want read", spec.Page, spec.Action)
		}
	}
}

func TestAttachCan_FillsNavGroups(t *testing.T) {
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleViewer, operator.GrantsFor(operator.RoleViewer))
	td := AttachCan(TemplateData{}, "/gui", p.Has)
	if !td.Can("dashboard", "read") {
		t.Fatal("viewer Can dashboard:read = false")
	}
	if td.Can("tenants", "read") {
		t.Fatal("viewer Can tenants:read = true")
	}
	if len(td.NavGroups) == 0 {
		t.Fatal("NavGroups empty")
	}
}

func flattenNav(groups []NavGroup) []string {
	var labels []string
	for _, g := range groups {
		for _, item := range g.Items {
			labels = append(labels, item.Label)
		}
	}
	return labels
}

func headingPresent(groups []NavGroup, heading string) bool {
	for _, g := range groups {
		if g.Heading == heading {
			return true
		}
	}
	return false
}

func labelPresent(groups []NavGroup, label string) bool {
	for _, g := range groups {
		for _, item := range g.Items {
			if item.Label == label {
				return true
			}
		}
	}
	return false
}
