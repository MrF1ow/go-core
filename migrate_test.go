package core

import "testing"

func TestIsForwardMigration(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"001_tenants_and_apps.sql", true},
		{"015_session_groups.sql", true},
		{"001_tenants_and_apps_rollback.sql", false},
		{"001_tenants_and_apps.down.sql", false},
		{"README.md", false},
		{"001_tenants_and_apps.sql.bak", false},
	}
	for _, tc := range cases {
		if got := isForwardMigration(tc.name); got != tc.want {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
