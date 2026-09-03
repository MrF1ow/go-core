package operator

import (
	"testing"

	"github.com/google/uuid"
)

func TestRestrictAppQuery_UnboundKeepsRequested(t *testing.T) {
	if RestrictAppQuery(nil, "requested-app") != "requested-app" {
		t.Fatal("unbound should keep requested")
	}
}

func TestRestrictAppQuery_BoundOverwritesRequested(t *testing.T) {
	bound := uuid.New()
	if RestrictAppQuery(&bound, uuid.New().String()) != bound.String() {
		t.Fatal("bound should overwrite requested")
	}
	if RestrictAppQuery(&bound, "") != bound.String() {
		t.Fatal("bound empty request should still return bound app")
	}
}

func TestForeignApp(t *testing.T) {
	bound := uuid.New()
	other := uuid.New()
	if !ForeignApp(&bound, other) {
		t.Fatal("foreign should be true for another app")
	}
	if ForeignApp(&bound, bound) {
		t.Fatal("foreign should be false for the bound app")
	}
	if ForeignApp(nil, other) {
		t.Fatal("foreign should be false when unbound")
	}
}

func TestForeignAppID_InvalidIsForeignWhenBound(t *testing.T) {
	bound := uuid.New()
	if !ForeignAppID(&bound, "not-a-uuid") {
		t.Fatal("invalid parse should be foreign when bound")
	}
	if ForeignAppID(nil, "not-a-uuid") {
		t.Fatal("invalid parse should not be foreign when unbound")
	}
	if ForeignAppID(&bound, bound.String()) {
		t.Fatal("bound id should not be foreign")
	}
}
