package admin

import (
	"testing"

	"github.com/google/uuid"
)

func TestSetDisabledAtRejectsNil(t *testing.T) {
	repo := &AccountRepository{}
	err := repo.SetDisabledAt(uuid.New(), nil)
	if err == nil {
		t.Fatal("SetDisabledAt(nil) succeeded")
	}
}
