package operator

import "testing"

func TestShouldLogAccess(t *testing.T) {
	tests := []struct {
		name string
		rec  AccessRecord
		want bool
	}{
		{
			name: "deny read api_key",
			rec: AccessRecord{
				Kind:     KindAPIKey,
				Decision: DecisionDeny,
				Action:   ActionRead,
			},
			want: true,
		},
		{
			name: "deny write",
			rec: AccessRecord{
				Kind:     KindAPIKey,
				Decision: DecisionDeny,
				Action:   ActionWrite,
			},
			want: true,
		},
		{
			name: "allow write api_key",
			rec: AccessRecord{
				Kind:     KindAPIKey,
				Decision: DecisionAllow,
				Action:   ActionWrite,
			},
			want: true,
		},
		{
			name: "allow read env_key",
			rec: AccessRecord{
				Kind:     KindEnvKey,
				Decision: DecisionAllow,
				Action:   ActionRead,
			},
			want: true,
		},
		{
			name: "allow read api_key viewer activity-logs",
			rec: AccessRecord{
				Kind:     KindAPIKey,
				Decision: DecisionAllow,
				Action:   ActionRead,
				Resource: ResLogs,
			},
			want: false,
		},
		{
			name: "allow read gui_account",
			rec: AccessRecord{
				Kind:     KindGUIAccount,
				Decision: DecisionAllow,
				Action:   ActionRead,
			},
			want: false,
		},
		{
			name: "garbage decision",
			rec: AccessRecord{
				Kind:     KindAPIKey,
				Decision: "maybe",
				Action:   ActionWrite,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldLogAccess(tt.rec); got != tt.want {
				t.Fatalf("ShouldLogAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}
