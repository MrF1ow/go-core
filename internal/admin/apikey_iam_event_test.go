package admin

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
)

func TestApiKeyCreate_SuperadminRecordsCreatePrincipal(t *testing.T) {
	got := apiKeyCreatePOST(t, superadminGUIPrincipal(), url.Values{
		"name":             {"root-key"},
		"key_type":         {KeyTypeAdmin},
		"operator_role_id": {operator.RoleIDSuperadmin.String()},
	})
	if got.response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got.response.Code, got.response.Body.String())
	}
	if len(got.events) != 1 {
		t.Fatalf("events = %d, want 1", len(got.events))
	}
	ev := got.events[0]
	if ev.Action != operator.ActionCreatePrincipal {
		t.Fatalf("action = %q", ev.Action)
	}
	if ev.ActorKind != string(operator.KindGUIAccount) {
		t.Fatalf("actor_kind = %q", ev.ActorKind)
	}
	if ev.TargetKind != operator.KindAPIKey {
		t.Fatalf("target_kind = %q", ev.TargetKind)
	}
	if ev.NewRoleID == nil || *ev.NewRoleID != operator.RoleIDSuperadmin {
		t.Fatalf("new_role_id = %v, want superadmin", ev.NewRoleID)
	}
	if ev.OldRoleID != nil {
		t.Fatalf("old_role_id = %v, want nil", ev.OldRoleID)
	}
	if got.created == nil || ev.TargetKeyID == nil || *ev.TargetKeyID != got.created.ID {
		t.Fatalf("target_key_id = %v, created = %v", ev.TargetKeyID, got.created)
	}
}

func TestApiKeyUpdate_RoleChangeRecordsAssign(t *testing.T) {
	current := operator.RoleIDViewer
	existing := &models.ApiKey{
		ID:             uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		KeyType:        KeyTypeAdmin,
		Name:           "ops",
		OperatorRoleID: &current,
	}
	got := apiKeyUpdatePUT(t, superadminGUIPrincipal(), existing, url.Values{
		"name":             {"ops"},
		"operator_role_id": {operator.RoleIDSupport.String()},
	})
	if got.response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got.response.Code, got.response.Body.String())
	}
	if len(got.events) != 1 {
		t.Fatalf("events = %d, want 1", len(got.events))
	}
	ev := got.events[0]
	if ev.Action != operator.ActionAssign {
		t.Fatalf("action = %q", ev.Action)
	}
	if ev.OldRoleID == nil || *ev.OldRoleID != operator.RoleIDViewer {
		t.Fatalf("old_role_id = %v, want viewer", ev.OldRoleID)
	}
	if ev.NewRoleID == nil || *ev.NewRoleID != operator.RoleIDSupport {
		t.Fatalf("new_role_id = %v, want support", ev.NewRoleID)
	}
}

func TestApiKeyUpdate_SameRoleRecordsNothing(t *testing.T) {
	current := operator.RoleIDSupport
	existing := &models.ApiKey{
		ID:             uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		KeyType:        KeyTypeAdmin,
		Name:           "ops",
		OperatorRoleID: &current,
	}
	got := apiKeyUpdatePUT(t, superadminGUIPrincipal(), existing, url.Values{
		"name":             {"ops-renamed"},
		"operator_role_id": {current.String()},
	})
	if got.response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got.response.Code, got.response.Body.String())
	}
	if len(got.events) != 0 {
		t.Fatalf("events = %#v, want none", got.events)
	}
}
