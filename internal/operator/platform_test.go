package operator

import "testing"

func TestPlatformResource_CatalogCoverage(t *testing.T) {
	wantApp := map[string]struct{}{
		ResUsers:       {},
		ResSessions:    {},
		ResLogs:        {},
		ResAPIKeys:     {},
		ResOAuth:       {},
		ResIPRules:     {},
		ResWebhooks:    {},
		ResEndUserRBAC: {},
	}
	wantPlatform := map[string]struct{}{
		ResDashboard:     {},
		ResTenants:       {},
		ResApplications:  {},
		ResOIDC:          {},
		ResSessionGroups: {},
		ResEmail:         {},
		ResMonitoring:    {},
		ResSettings:      {},
		ResAdminIAM:      {},
	}

	seen := map[string]struct{}{}
	for _, p := range Catalog() {
		if _, ok := seen[p.Resource]; ok {
			continue
		}
		seen[p.Resource] = struct{}{}
		got := PlatformResource(p.Resource)
		_, app := wantApp[p.Resource]
		_, plat := wantPlatform[p.Resource]
		if app == plat {
			t.Errorf("catalog resource %q must sit in exactly one expected set", p.Resource)
			continue
		}
		if app && got {
			t.Errorf("%q classified as platform, want app-scoped", p.Resource)
		}
		if plat && !got {
			t.Errorf("%q classified as app-scoped, want platform", p.Resource)
		}
	}
	for r := range wantApp {
		if _, ok := seen[r]; !ok {
			t.Errorf("app-scoped %q missing from catalog", r)
		}
	}
	for r := range wantPlatform {
		if _, ok := seen[r]; !ok {
			t.Errorf("platform %q missing from catalog", r)
		}
	}
}

func TestPlatformResource_UnknownIsPlatform(t *testing.T) {
	if !PlatformResource("not-a-resource") {
		t.Fatal("unknown resource must be platform")
	}
	if !PlatformResource("") {
		t.Fatal("empty resource must be platform")
	}
}
