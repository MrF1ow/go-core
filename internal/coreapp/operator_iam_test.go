package coreapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
)

type iamEventMem struct {
	mu     sync.Mutex
	events []operator.IAMEvent
}

func (m *iamEventMem) list(_ context.Context, limit int32, targetKeyID, targetAccountID *uuid.UUID) ([]operator.IAMEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]operator.IAMEvent, 0)
	for _, ev := range m.events {
		if targetKeyID != nil && (ev.TargetKeyID == nil || *ev.TargetKeyID != *targetKeyID) {
			continue
		}
		if targetAccountID != nil && (ev.TargetAccountID == nil || *ev.TargetAccountID != *targetAccountID) {
			continue
		}
		out = append(out, ev)
		if int32(len(out)) >= limit {
			break
		}
	}
	return out, nil
}

func (m *iamEventMem) write(ev operator.IAMEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append([]operator.IAMEvent{ev}, m.events...)
	return nil
}

type iamEventHarness struct {
	engine      *gin.Engine
	mem         *iamEventMem
	viewerKeyID uuid.UUID
	keys        map[uuid.UUID]*models.ApiKey
}

func iamEventTestEngine(t *testing.T) *iamEventHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store := &accessLogKeyStore{}
	viewerRole := operator.RoleIDViewer
	superRole := operator.RoleIDSuperadmin
	viewerKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &viewerRole,
	}
	superKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &superRole,
	}
	store.put(accessViewerKey, viewerKey)
	store.put(accessSuperadminKey, superKey)
	keys := map[uuid.UUID]*models.ApiKey{
		viewerKey.ID: viewerKey,
		superKey.ID:  superKey,
	}
	grants := &accessLogGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		viewerRole:             {name: operator.RoleViewer, keys: operator.GrantsFor(operator.RoleViewer)},
		superRole:              {name: operator.RoleSuperadmin, keys: operator.GrantsFor(operator.RoleSuperadmin)},
		operator.RoleIDSupport: {name: operator.RoleSupport, keys: operator.GrantsFor(operator.RoleSupport)},
	}}

	mem := &iamEventMem{}
	handler := &admin.Handler{
		IAMEventList:  mem.list,
		IAMEventWrite: mem.write,
		GetAPIKey: func(id string) (*models.ApiKey, error) {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return nil, err
			}
			key, ok := keys[parsed]
			if !ok {
				return nil, pgx.ErrNoRows
			}
			copied := *key
			if key.OperatorRoleID != nil {
				role := *key.OperatorRoleID
				copied.OperatorRoleID = &role
			}
			return &copied, nil
		},
		UpdateAPIKeyRole: func(id string, roleID *uuid.UUID) error {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return err
			}
			key, ok := keys[parsed]
			if !ok {
				return pgx.ErrNoRows
			}
			if roleID == nil {
				key.OperatorRoleID = nil
				return nil
			}
			copied := *roleID
			key.OperatorRoleID = &copied
			return nil
		},
	}
	engine := gin.New()
	group := engine.Group("/admin")
	group.Use(middleware.AdminAuthMiddleware(accessEnvKey, store, grants))
	group.GET("/operator/iam-events", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorIAMEvents)
	group.PUT("/operator/keys/:id/role", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorKeyRole)
	return &iamEventHarness{
		engine:      engine,
		mem:         mem,
		viewerKeyID: viewerKey.ID,
		keys:        keys,
	}
}

func iamEventDo(engine *gin.Engine, method, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Admin-API-Key", key)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func iamEventList(t *testing.T, engine *gin.Engine) []operator.IAMEvent {
	t.Helper()
	rec := accessLogDo(engine, http.MethodGet, "/admin/operator/iam-events", accessSuperadminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries []operator.IAMEvent `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Entries
}

func TestOperatorIAMEvents_ViewerForbidden(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := accessLogDo(h.engine, http.MethodGet, "/admin/operator/iam-events", accessViewerKey)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestOperatorIAMEvents_SuperadminListsNewestFirst(t *testing.T) {
	h := iamEventTestEngine(t)
	older := operator.IAMEvent{Action: operator.ActionCreatePrincipal, ActorKind: string(operator.KindAPIKey)}
	newer := operator.IAMEvent{Action: operator.ActionAssign, ActorKind: string(operator.KindAPIKey)}
	if err := h.mem.write(older); err != nil {
		t.Fatal(err)
	}
	if err := h.mem.write(newer); err != nil {
		t.Fatal(err)
	}
	entries := iamEventList(t, h.engine)
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Action != operator.ActionAssign {
		t.Fatalf("first action = %q, want assign", entries[0].Action)
	}
	if entries[1].Action != operator.ActionCreatePrincipal {
		t.Fatalf("second action = %q, want create_principal", entries[1].Action)
	}
}

func TestOperatorIAMEvents_InvalidTargetKeyID(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := accessLogDo(h.engine, http.MethodGet, "/admin/operator/iam-events?target_key_id=not-a-uuid", accessSuperadminKey)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestOperatorKeyRole_SuperadminAssignsViewerToSupport(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"operator_role_id":"` + operator.RoleIDSupport.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/keys/"+h.viewerKeyID.String()+"/role", accessSuperadminKey, body)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	entries := iamEventList(t, h.engine)
	if len(entries) != 1 {
		t.Fatalf("events = %d, want 1", len(entries))
	}
	ev := entries[0]
	if ev.Action != operator.ActionAssign {
		t.Fatalf("action = %q", ev.Action)
	}
	if ev.OldRoleID == nil || *ev.OldRoleID != operator.RoleIDViewer {
		t.Fatalf("old_role_id = %v, want viewer", ev.OldRoleID)
	}
	if ev.NewRoleID == nil || *ev.NewRoleID != operator.RoleIDSupport {
		t.Fatalf("new_role_id = %v, want support", ev.NewRoleID)
	}
	stored := h.keys[h.viewerKeyID]
	if stored.OperatorRoleID == nil || *stored.OperatorRoleID != operator.RoleIDSupport {
		t.Fatalf("stored role = %v, want support", stored.OperatorRoleID)
	}
}

func TestOperatorKeyRole_ViewerForbidden(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"operator_role_id":"` + operator.RoleIDSupport.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/keys/"+h.viewerKeyID.String()+"/role", accessViewerKey, body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(iamEventList(t, h.engine)) != 0 {
		t.Fatal("viewer PUT must not write an IAM event")
	}
	stored := h.keys[h.viewerKeyID]
	if stored.OperatorRoleID == nil || *stored.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("stored role = %v, want viewer", stored.OperatorRoleID)
	}
}
