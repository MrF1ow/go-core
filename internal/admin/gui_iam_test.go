package admin

import (
	"testing"

	"github.com/MrF1ow/go-core/internal/operator"
)

func TestRosterStatus(t *testing.T) {
	disabled := true
	enabled := false
	cases := []struct {
		name  string
		entry operator.RosterEntry
		want  string
	}{
		{name: "revoked wins", entry: operator.RosterEntry{Revoked: true, Disabled: &disabled}, want: "Revoked"},
		{name: "disabled pointer true", entry: operator.RosterEntry{Disabled: &disabled}, want: "Disabled"},
		{name: "disabled pointer false", entry: operator.RosterEntry{Disabled: &enabled}, want: "Active"},
		{name: "nil disabled", entry: operator.RosterEntry{}, want: "Active"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rosterStatus(tc.entry); got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}
