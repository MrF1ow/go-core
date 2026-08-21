package admin

import (
	"testing"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
)

func TestRoleIDForSetupAccount(t *testing.T) {
	tests := []struct {
		name          string
		existingCount int64
		want          string
	}{
		{name: "first account", existingCount: 0, want: operator.RoleIDSuperadmin.String()},
		{name: "additional account", existingCount: 1, want: operator.RoleIDViewer.String()},
		{name: "many existing accounts", existingCount: 12, want: operator.RoleIDViewer.String()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := RoleIDForSetupAccount(test.existingCount)
			if got.String() != test.want {
				t.Fatalf("role ID = %s, want %s", got, test.want)
			}
		})
	}
}

func TestAccountRepositoryCreateRejectsMissingOperatorRole(t *testing.T) {
	repository := &AccountRepository{}
	err := repository.Create(&models.AdminAccount{})
	if err == nil {
		t.Fatal("Create accepted an account without an operator role")
	}
}
