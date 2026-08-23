package admin

import (
	"testing"

	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
)

func TestParseOptionalUUIDQuery(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	got, err := parseOptionalUUIDQuery(id.String())
	if err != nil || got == nil || *got != id {
		t.Fatalf("valid = %v, %v", got, err)
	}
	got, err = parseOptionalUUIDQuery("")
	if err != nil || got != nil {
		t.Fatalf("empty = %v, %v", got, err)
	}
	if _, err = parseOptionalUUIDQuery("not-a-uuid"); err == nil {
		t.Fatal("invalid UUID succeeded")
	}
}

func TestParseOptionalFormAppID(t *testing.T) {
	id := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	got, err := parseOptionalFormAppID("  " + id.String() + "  ")
	if err != nil || got == nil || *got != id {
		t.Fatalf("valid = %v, %v", got, err)
	}
	got, err = parseOptionalFormAppID("")
	if err != nil || got != nil {
		t.Fatalf("empty = %v, %v", got, err)
	}
	if _, err = parseOptionalFormAppID("not-a-uuid"); err == nil {
		t.Fatal("invalid UUID succeeded")
	}
}

func TestParseDecisionQuery(t *testing.T) {
	got, err := parseDecisionQuery(operator.DecisionDeny)
	if err != nil || got == nil || *got != operator.DecisionDeny {
		t.Fatalf("deny = %v, %v", got, err)
	}
	got, err = parseDecisionQuery("")
	if err != nil || got != nil {
		t.Fatalf("empty = %v, %v", got, err)
	}
	if _, err = parseDecisionQuery("maybe"); err == nil {
		t.Fatal("invalid decision succeeded")
	}
}

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
