package coreapp

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
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
	engine         *gin.Engine
	handler        *admin.Handler
	mem            *iamEventMem
	store          *accessLogKeyStore
	viewerKeyID    uuid.UUID
	keys           map[uuid.UUID]*models.ApiKey
	superAccountID uuid.UUID
	accounts       map[uuid.UUID]*models.AdminAccount
	customRoleID   uuid.UUID
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
	adminRole := operator.RoleIDAdmin
	supportRole := operator.RoleIDSupport
	adminKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &adminRole,
	}
	supportKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &supportRole,
	}
	store.put(accessAdminKey, adminKey)
	store.put(accessSupportKey, supportKey)
	keys := map[uuid.UUID]*models.ApiKey{
		viewerKey.ID:  viewerKey,
		superKey.ID:   superKey,
		adminKey.ID:   adminKey,
		supportKey.ID: supportKey,
	}
	grants := &accessLogGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		viewerRole:             {name: operator.RoleViewer, keys: operator.GrantsFor(operator.RoleViewer)},
		superRole:              {name: operator.RoleSuperadmin, keys: operator.GrantsFor(operator.RoleSuperadmin)},
		operator.RoleIDSupport: {name: operator.RoleSupport, keys: operator.GrantsFor(operator.RoleSupport)},
		operator.RoleIDAdmin:   {name: operator.RoleAdmin, keys: operator.GrantsFor(operator.RoleAdmin)},
	}}

	mem := &iamEventMem{}
	customRoleID := uuid.New()
	handler := &admin.Handler{
		IAMEventList:  mem.list,
		IAMEventWrite: mem.write,
		RoleExists: func(id uuid.UUID) (bool, error) {
			return id == customRoleID, nil
		},
		GetOperatorRole: func(_ context.Context, id uuid.UUID) (sqlcgen.OperatorRole, error) {
			if operator.IsSystemRoleID(id) {
				return sqlcgen.OperatorRole{ID: id, Name: operator.SystemRoleName(id), IsSystem: true}, nil
			}
			if id == customRoleID {
				return sqlcgen.OperatorRole{ID: id, Name: "auditor", IsSystem: false}, nil
			}
			return sqlcgen.OperatorRole{}, pgx.ErrNoRows
		},
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
		CreateAPIKey: func(key *models.ApiKey) error {
			if key.ID == uuid.Nil {
				key.ID = uuid.New()
			}
			stored := *key
			if key.OperatorRoleID != nil {
				role := *key.OperatorRoleID
				stored.OperatorRoleID = &role
			}
			if key.ExpiresAt != nil {
				exp := *key.ExpiresAt
				stored.ExpiresAt = &exp
			}
			if key.AppID != nil {
				appID := *key.AppID
				stored.AppID = &appID
			}
			keys[stored.ID] = &stored
			return nil
		},
	}
	accounts := map[uuid.UUID]*models.AdminAccount{}
	superAccount := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "root",
		Email:          "root@example.com",
		OperatorRoleID: operator.RoleIDSuperadmin,
	}
	accounts[superAccount.ID] = superAccount
	handler.CreateAccount = func(account *models.AdminAccount) error {
		if account.ID == uuid.Nil {
			account.ID = uuid.New()
		}
		copied := *account
		accounts[copied.ID] = &copied
		*account = copied
		return nil
	}
	handler.GetAccount = func(id string) (*models.AdminAccount, error) {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		account, ok := accounts[parsed]
		if !ok {
			return nil, nil
		}
		copied := *account
		if account.DisabledAt != nil {
			disabled := *account.DisabledAt
			copied.DisabledAt = &disabled
		}
		return &copied, nil
	}
	handler.UpdateAccountRole = func(id uuid.UUID, roleID uuid.UUID) error {
		account, ok := accounts[id]
		if !ok {
			return pgx.ErrNoRows
		}
		account.OperatorRoleID = roleID
		return nil
	}
	handler.DisableAccount = func(id uuid.UUID) error {
		account, ok := accounts[id]
		if !ok {
			return pgx.ErrNoRows
		}
		now := time.Now().UTC()
		account.DisabledAt = &now
		return nil
	}
	handler.CountEnabledSuperadmins = func() (int64, error) {
		var n int64
		for _, account := range accounts {
			if account.DisabledAt == nil && account.OperatorRoleID == operator.RoleIDSuperadmin {
				n++
			}
		}
		return n, nil
	}
	engine := gin.New()
	group := engine.Group("/admin")
	group.Use(middleware.AdminAuthMiddleware(accessEnvKey, store, grants))
	group.GET("/operator/iam-events", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorIAMEvents)
	group.GET("/operator/iam-events/export", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorIAMEventsExport)
	group.PUT("/operator/keys/:id/role", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorKeyRole)
	group.POST("/operator/keys", requireOp(operator.ResAPIKeys, operator.ActionWrite), handler.OperatorCreateKey)
	group.GET("/tenants", requireOp(operator.ResTenants, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	group.POST("/operator/accounts", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorCreateAccount)
	group.PUT("/operator/accounts/:id/role", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorAccountRole)
	group.POST("/operator/accounts/:id/disable", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorDisableAccount)
	group.GET("/operator/roles", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorListRoles)
	group.POST("/operator/roles", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorCreateRole)
	group.PUT("/operator/roles/:id", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorUpdateRole)
	group.PUT("/operator/roles/:id/permissions", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorReplaceRolePermissions)
	group.DELETE("/operator/roles/:id", requireOp(operator.ResAdminIAM, operator.ActionWrite), handler.OperatorDeleteRole)
	return &iamEventHarness{
		engine:         engine,
		handler:        handler,
		mem:            mem,
		store:          store,
		viewerKeyID:    viewerKey.ID,
		keys:           keys,
		superAccountID: superAccount.ID,
		accounts:       accounts,
		customRoleID:   customRoleID,
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

func TestOperatorIAMEventsExport_ViewerForbiddenJSON(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := accessLogDo(h.engine, http.MethodGet, "/admin/operator/iam-events/export", accessViewerKey)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestOperatorIAMEventsExport_SuperadminCSV(t *testing.T) {
	h := iamEventTestEngine(t)
	keyID := uuid.New()
	if err := h.mem.write(operator.IAMEvent{
		Action:      operator.ActionAssign,
		ActorKind:   string(operator.KindAPIKey),
		TargetKind:  operator.KindAPIKey,
		TargetKeyID: &keyID,
	}); err != nil {
		t.Fatal(err)
	}
	rec := accessLogDo(h.engine, http.MethodGet, "/admin/operator/iam-events/export", accessSuperadminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Export-Truncated") != "false" {
		t.Fatalf("truncated = %q", rec.Header().Get("X-Export-Truncated"))
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "operator-iam-events.csv") {
		t.Fatalf("disposition = %q", rec.Header().Get("Content-Disposition"))
	}
	body := rec.Body.Bytes()
	if len(body) >= 3 && body[0] == 0xef && body[1] == 0xbb && body[2] == 0xbf {
		t.Fatal("utf-8 BOM present")
	}
	rows, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := []string{
		"id", "at", "actor_kind", "actor_key_id", "actor_account_id",
		"target_kind", "target_key_id", "target_account_id", "old_role_id", "new_role_id", "action",
	}
	if len(rows) < 2 {
		t.Fatalf("csv rows = %#v", rows)
	}
	if strings.Join(rows[0], ",") != strings.Join(wantHeader, ",") {
		t.Fatalf("header = %#v", rows[0])
	}
	found := false
	for _, row := range rows[1:] {
		if len(row) != len(wantHeader) {
			t.Fatalf("row width = %d, want %d: %#v", len(row), len(wantHeader), row)
		}
		if row[10] == operator.ActionAssign && row[6] == keyID.String() {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing assign row in %#v", rows)
	}
}

func TestOperatorIAMEventsExport_TruncatesAtExportMaxRows(t *testing.T) {
	h := iamEventTestEngine(t)
	for i := 0; i < operator.ExportMaxRows+1; i++ {
		if err := h.mem.write(operator.IAMEvent{Action: operator.ActionAssign, ActorKind: string(operator.KindAPIKey)}); err != nil {
			t.Fatal(err)
		}
	}
	rec := accessLogDo(h.engine, http.MethodGet, "/admin/operator/iam-events/export", accessSuperadminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Export-Truncated") != "true" {
		t.Fatalf("truncated = %q", rec.Header().Get("X-Export-Truncated"))
	}
	rows, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != operator.ExportMaxRows+1 {
		t.Fatalf("csv rows = %d, want %d (header + cap)", len(rows), operator.ExportMaxRows+1)
	}
}

func TestOperatorIAMEvents_ListLimitStillCapsAt1000(t *testing.T) {
	h := iamEventTestEngine(t)
	for i := 0; i < 1500; i++ {
		if err := h.mem.write(operator.IAMEvent{Action: operator.ActionAssign, ActorKind: string(operator.KindAPIKey)}); err != nil {
			t.Fatal(err)
		}
	}
	rec := accessLogDo(h.engine, http.MethodGet, "/admin/operator/iam-events?limit=5000", accessSuperadminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries []operator.IAMEvent `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != 1000 {
		t.Fatalf("len = %d, want 1000", len(body.Entries))
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

func TestOperatorKeyRole_EmptyRoleIsBadRequest(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/keys/"+h.viewerKeyID.String()+"/role", accessSuperadminKey, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stored := h.keys[h.viewerKeyID]
	if stored.OperatorRoleID == nil || *stored.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("stored role = %v, want viewer", stored.OperatorRoleID)
	}
	if len(iamEventList(t, h.engine)) != 0 {
		t.Fatal("empty role PUT must not write an IAM event")
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

func TestOperatorCreateAccount_SuperadminCreatesViewer(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"username":"ops-viewer","email":"ops@example.com","password":"twelvechars!!"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts", accessSuperadminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got struct {
		ID             uuid.UUID `json:"id"`
		OperatorRoleID uuid.UUID `json:"operator_role_id"`
		Role           string    `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("operator_role_id = %s, want viewer", got.OperatorRoleID)
	}
	if got.Role != operator.RoleViewer {
		t.Fatalf("role = %q, want viewer", got.Role)
	}
	found := false
	for _, ev := range iamEventList(t, h.engine) {
		if ev.Action == operator.ActionCreatePrincipal && ev.TargetAccountID != nil && *ev.TargetAccountID == got.ID {
			if ev.NewRoleID == nil || *ev.NewRoleID != operator.RoleIDViewer {
				t.Fatalf("new_role_id = %v, want viewer", ev.NewRoleID)
			}
			if ev.TargetKind != operator.KindGUIAccount {
				t.Fatalf("target_kind = %q", ev.TargetKind)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected create_principal event")
	}
}

func TestOperatorCreateAccount_ViewerForbidden(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"username":"ops-viewer","email":"ops@example.com","password":"twelvechars!!"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts", accessViewerKey, body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(iamEventList(t, h.engine)) != 0 {
		t.Fatal("viewer POST must not write an IAM event")
	}
}

func TestOperatorAccountRole_LastSuperadminDemoteConflict(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"operator_role_id":"` + operator.RoleIDAdmin.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/accounts/"+h.superAccountID.String()+"/role", accessSuperadminKey, body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stored := h.accounts[h.superAccountID]
	if stored.OperatorRoleID != operator.RoleIDSuperadmin {
		t.Fatalf("role = %s, want superadmin", stored.OperatorRoleID)
	}
	if len(iamEventList(t, h.engine)) != 0 {
		t.Fatal("last-superadmin demote must not write an IAM event")
	}
}

func TestOperatorDisableAccount_LastSuperadminConflict(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts/"+h.superAccountID.String()+"/disable", accessSuperadminKey, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if h.accounts[h.superAccountID].DisabledAt != nil {
		t.Fatal("last superadmin was disabled")
	}
	if len(iamEventList(t, h.engine)) != 0 {
		t.Fatal("last-superadmin disable must not write an IAM event")
	}
}

func TestOperatorDisableAccount_ViewerIdempotent(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"username":"ops-viewer","email":"ops@example.com","password":"twelvechars!!"}`
	created := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts", accessSuperadminKey, body)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	var got struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	first := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts/"+got.ID.String()+"/disable", accessSuperadminKey, "")
	if first.Code != http.StatusNoContent {
		t.Fatalf("first disable status = %d, body = %s", first.Code, first.Body.String())
	}
	second := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts/"+got.ID.String()+"/disable", accessSuperadminKey, "")
	if second.Code != http.StatusNoContent {
		t.Fatalf("second disable status = %d, body = %s", second.Code, second.Body.String())
	}
	disableCount := 0
	for _, ev := range iamEventList(t, h.engine) {
		if ev.Action == operator.ActionDisablePrincipal {
			disableCount++
		}
	}
	if disableCount != 1 {
		t.Fatalf("disable events = %d, want 1", disableCount)
	}
}

func TestOperatorKeyRole_SuperadminStampsExistingCustomRole(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"operator_role_id":"` + h.customRoleID.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/keys/"+h.viewerKeyID.String()+"/role", accessSuperadminKey, body)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stored := h.keys[h.viewerKeyID]
	if stored.OperatorRoleID == nil || *stored.OperatorRoleID != h.customRoleID {
		t.Fatalf("stored role = %v, want custom", stored.OperatorRoleID)
	}
}

func TestOperatorKeyRole_RandomUUIDRejected(t *testing.T) {
	h := iamEventTestEngine(t)
	unknown := uuid.New()
	body := `{"operator_role_id":"` + unknown.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/keys/"+h.viewerKeyID.String()+"/role", accessSuperadminKey, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stored := h.keys[h.viewerKeyID]
	if stored.OperatorRoleID == nil || *stored.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("stored role = %v, want viewer", stored.OperatorRoleID)
	}
}

func TestOperatorKeyRole_ViewerStampCustomForbidden(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"operator_role_id":"` + h.customRoleID.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/keys/"+h.viewerKeyID.String()+"/role", accessViewerKey, body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stored := h.keys[h.viewerKeyID]
	if stored.OperatorRoleID == nil || *stored.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("stored role = %v, want viewer", stored.OperatorRoleID)
	}
}

func TestOperatorAccountRole_StampsExistingCustomRole(t *testing.T) {
	h := iamEventTestEngine(t)
	created := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts", accessSuperadminKey, `{"username":"ops-custom","email":"ops@example.com","password":"twelvechars!!"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	var got struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	body := `{"operator_role_id":"` + h.customRoleID.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/accounts/"+got.ID.String()+"/role", accessSuperadminKey, body)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stored := h.accounts[got.ID]
	if stored.OperatorRoleID != h.customRoleID {
		t.Fatalf("role = %s, want custom", stored.OperatorRoleID)
	}
}

func TestOperatorAccountRole_EmptyRoleIsBadRequest(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/accounts/"+h.superAccountID.String()+"/role", accessSuperadminKey, `{"operator_role_id":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stored := h.accounts[h.superAccountID]
	if stored.OperatorRoleID != operator.RoleIDSuperadmin {
		t.Fatalf("role = %s, want superadmin", stored.OperatorRoleID)
	}
}

func TestOperatorAccountRole_LastSuperadminStillJSONConflict(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"operator_role_id":"` + operator.RoleIDAdmin.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/accounts/"+h.superAccountID.String()+"/role", accessSuperadminKey, body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	if stored := h.accounts[h.superAccountID]; stored.OperatorRoleID != operator.RoleIDSuperadmin {
		t.Fatalf("role = %s, want superadmin", stored.OperatorRoleID)
	}
}

func TestOperatorDisableAccount_LastSuperadminTriggerConflict(t *testing.T) {
	h := iamEventTestEngine(t)
	second := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "root-peer",
		OperatorRoleID: operator.RoleIDSuperadmin,
	}
	h.accounts[second.ID] = second
	h.handler.DisableAccount = func(uuid.UUID) error {
		return &pgconn.PgError{Code: "P0001", Message: "cannot demote or disable the last enabled superadmin"}
	}
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/accounts/"+h.superAccountID.String()+"/disable", accessSuperadminKey, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cannot demote or disable the last enabled superadmin") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if h.accounts[h.superAccountID].DisabledAt != nil {
		t.Fatal("last superadmin was disabled")
	}
	if len(iamEventList(t, h.engine)) != 0 {
		t.Fatal("trigger last-superadmin disable must not write an IAM event")
	}
}

func TestOperatorAccountRole_LastSuperadminTriggerConflict(t *testing.T) {
	h := iamEventTestEngine(t)
	second := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "root-peer",
		OperatorRoleID: operator.RoleIDSuperadmin,
	}
	h.accounts[second.ID] = second
	h.handler.UpdateAccountRole = func(uuid.UUID, uuid.UUID) error {
		return &pgconn.PgError{Code: "P0001", Message: "cannot demote or disable the last enabled superadmin"}
	}
	body := `{"operator_role_id":"` + operator.RoleIDAdmin.String() + `"}`
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/accounts/"+h.superAccountID.String()+"/role", accessSuperadminKey, body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cannot demote or disable the last enabled superadmin") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if h.accounts[h.superAccountID].OperatorRoleID != operator.RoleIDSuperadmin {
		t.Fatalf("role = %s, want superadmin", h.accounts[h.superAccountID].OperatorRoleID)
	}
	if len(iamEventList(t, h.engine)) != 0 {
		t.Fatal("trigger last-superadmin demote must not write an IAM event")
	}
}

func TestOperatorCreateRole_AdminIAMGrantRejected(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"name":"auditor","grants":["users:read","admin_iam:write"]}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/roles", accessSuperadminKey, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), operator.ErrAdminIAMOnCustomRole.Error()) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestOperatorCreateRole_ReservedSystemNameRejected(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/roles", accessSuperadminKey, `{"name":"admin","grants":["users:read"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), operator.ErrReservedSystemRoleName.Error()) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestOperatorCreateRole_CreatesCustom(t *testing.T) {
	h := iamEventTestEngine(t)
	created := uuid.New()
	h.handler.CreateOperatorRole = func(_ context.Context, name, description string, grants []operator.Permission) (sqlcgen.OperatorRole, error) {
		if name != "auditor" {
			t.Fatalf("name = %q", name)
		}
		if len(grants) != 1 || grants[0].Resource != operator.ResUsers || grants[0].Action != operator.ActionRead {
			t.Fatalf("grants = %#v", grants)
		}
		return sqlcgen.OperatorRole{ID: created, Name: name, Description: description, IsSystem: false}, nil
	}
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/roles", accessSuperadminKey, `{"name":"auditor","grants":["users:read"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got struct {
		ID       uuid.UUID `json:"id"`
		Name     string    `json:"name"`
		IsSystem bool      `json:"is_system"`
		Grants   []string  `json:"grants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != created || got.Name != "auditor" || got.IsSystem {
		t.Fatalf("got %#v", got)
	}
	if len(got.Grants) != 1 || got.Grants[0] != operator.ResUsers+":"+operator.ActionRead {
		t.Fatalf("grants = %#v", got.Grants)
	}
}

func TestOperatorUpdateRole_SystemImmutable(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/roles/"+operator.RoleIDSuperadmin.String(), accessSuperadminKey, `{"name":"not-superadmin"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), operator.ErrSystemRoleImmutable.Error()) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestOperatorReplaceRolePermissions_SystemImmutable(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := iamEventDo(h.engine, http.MethodPut, "/admin/operator/roles/"+operator.RoleIDAdmin.String()+"/permissions", accessSuperadminKey, `{"grants":["users:read"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), operator.ErrSystemRoleImmutable.Error()) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestOperatorDeleteRole_SystemImmutable(t *testing.T) {
	h := iamEventTestEngine(t)
	rec := iamEventDo(h.engine, http.MethodDelete, "/admin/operator/roles/"+operator.RoleIDViewer.String(), accessSuperadminKey, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), operator.ErrSystemRoleImmutable.Error()) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func operatorCreateKeyExpiry() string {
	return time.Now().UTC().Add(90 * 24 * time.Hour).Format("2006-01-02T15:04")
}

func TestOperatorCreateKey_Superadmin201SecretOnce(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"name":"ci","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got createKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Secret, "ak_") {
		t.Fatalf("secret = %q", got.Secret)
	}
	if got.KeyType != admin.KeyTypeAdmin {
		t.Fatalf("key_type = %q", got.KeyType)
	}
	if got.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("role = %s, want viewer", got.OperatorRoleID)
	}
	stored, ok := h.keys[got.ID]
	if !ok {
		t.Fatal("created key missing from store")
	}
	if stored.AppID != nil {
		t.Fatalf("app_id = %v, want nil", stored.AppID)
	}
	if stored.KeyHash == got.Secret {
		t.Fatal("stored hash is the raw secret")
	}
	replay, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(replay), got.Secret) {
		t.Fatal("stored key JSON still contains the raw secret")
	}
	entries := iamEventList(t, h.engine)
	if len(entries) != 1 || entries[0].Action != operator.ActionCreatePrincipal {
		t.Fatalf("events = %#v", entries)
	}
}

func TestOperatorCreateKey_MissingExpiry400(t *testing.T) {
	h := iamEventTestEngine(t)
	before := len(h.keys)
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, `{"name":"ci"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(h.keys) != before {
		t.Fatal("missing expiry still inserted a key")
	}
}

func TestOperatorCreateKey_EmptyRoleIsViewer(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"name":"ci","operator_role_id":"","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got createKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("role = %s, want viewer", got.OperatorRoleID)
	}
}

func TestOperatorCreateKey_SupportForbidden(t *testing.T) {
	h := iamEventTestEngine(t)
	before := len(h.keys)
	body := `{"name":"ci","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSupportKey, body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(h.keys) != before {
		t.Fatal("support mint still inserted a key")
	}
}

func TestOperatorCreateKey_AdminMintsViewer(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"name":"ci","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessAdminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got createKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("role = %s, want viewer", got.OperatorRoleID)
	}
}

func TestOperatorCreateKey_AdminCannotStampNonViewer(t *testing.T) {
	h := iamEventTestEngine(t)
	before := len(h.keys)
	body := `{"name":"ci","operator_role_id":"` + operator.RoleIDAdmin.String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessAdminKey, body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(h.keys) != before {
		t.Fatal("denied stamp still inserted a key")
	}
}

func TestOperatorCreateKey_IgnoresAppTypeAndPersistsAppID(t *testing.T) {
	h := iamEventTestEngine(t)
	appID := uuid.New()
	body := `{"name":"ci","key_type":"app","app_id":"` + appID.String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got createKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KeyType != admin.KeyTypeAdmin {
		t.Fatalf("key_type = %q", got.KeyType)
	}
	if got.AppID == nil || *got.AppID != appID {
		t.Fatalf("response app_id = %v, want %s", got.AppID, appID)
	}
	stored := h.keys[got.ID]
	if stored.KeyType != admin.KeyTypeAdmin {
		t.Fatalf("stored key_type = %q", stored.KeyType)
	}
	if stored.AppID == nil || *stored.AppID != appID {
		t.Fatalf("stored app_id = %v, want %s", stored.AppID, appID)
	}
}

func TestOperatorCreateKey_EmptyAppIDStaysNull(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"name":"ci","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got createKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.AppID != nil {
		t.Fatalf("response app_id = %v, want nil", got.AppID)
	}
	if h.keys[got.ID].AppID != nil {
		t.Fatalf("stored app_id = %v, want nil", h.keys[got.ID].AppID)
	}
}

func TestOperatorCreateKey_SuperadminAppIDRejected(t *testing.T) {
	h := iamEventTestEngine(t)
	before := len(h.keys)
	body := `{"name":"ci","operator_role_id":"` + operator.RoleIDSuperadmin.String() + `","app_id":"` + uuid.New().String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(h.keys) != before {
		t.Fatal("superadmin bind still inserted a key")
	}
}

func TestOperatorCreateKey_InvalidAppIDRejected(t *testing.T) {
	h := iamEventTestEngine(t)
	before := len(h.keys)
	body := `{"name":"ci","app_id":"not-a-uuid","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(h.keys) != before {
		t.Fatal("invalid app_id still inserted a key")
	}
}

func TestOperatorCreateKey_BoundMintForbidden(t *testing.T) {
	h := iamEventTestEngine(t)
	boundApp := uuid.New()
	adminRole := operator.RoleIDAdmin
	boundKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &adminRole,
		AppID:          &boundApp,
	}
	h.store.put("ak_bound_mint_forbidden", boundKey)
	before := len(h.keys)
	body := `{"name":"ci","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", "ak_bound_mint_forbidden", body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(h.keys) != before {
		t.Fatal("bound mint inserted a key")
	}
}

func TestBoundAdminKeyRestrict(t *testing.T) {
	err := &pgconn.PgError{Code: "P0001", Message: "application has bound admin keys"}
	if !admin.BoundAdminKeyRestrict(err) {
		t.Fatal("pg raise should match")
	}
	if admin.BoundAdminKeyRestrict(errors.New("Failed to delete application")) {
		t.Fatal("generic error should not match")
	}
}

func TestLastSuperadminRestrict(t *testing.T) {
	err := &pgconn.PgError{Code: "P0001", Message: "cannot demote or disable the last enabled superadmin"}
	if !admin.LastSuperadminRestrict(err) {
		t.Fatal("pg raise should match")
	}
	if admin.LastSuperadminRestrict(errors.New("Failed to disable operator account")) {
		t.Fatal("generic error should not match")
	}
}

type createKeyResponse struct {
	ID             uuid.UUID  `json:"id"`
	KeyType        string     `json:"key_type"`
	OperatorRoleID uuid.UUID  `json:"operator_role_id"`
	AppID          *uuid.UUID `json:"app_id"`
	Secret         string     `json:"secret"`
}

func TestOperatorCreateKey_MintedKeyAuthAndRevoke(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"name":"ci","operator_role_id":"` + operator.RoleIDSuperadmin.String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got createKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	stored := h.keys[got.ID]
	h.store.put(got.Secret, stored)
	auth := iamEventDo(h.engine, http.MethodGet, "/admin/tenants", got.Secret, "")
	if auth.Code != http.StatusOK {
		t.Fatalf("minted auth status = %d, body = %s", auth.Code, auth.Body.String())
	}
	stored.IsRevoked = true
	revoked := iamEventDo(h.engine, http.MethodGet, "/admin/tenants", got.Secret, "")
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked auth status = %d, body = %s", revoked.Code, revoked.Body.String())
	}
	if _, ok := h.keys[got.ID]; !ok {
		t.Fatal("revoked key row was dropped")
	}
}

func TestOperatorCreateKey_ViewerMintNever401OnTenants(t *testing.T) {
	h := iamEventTestEngine(t)
	body := `{"name":"ci","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got createKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	h.store.put(got.Secret, h.keys[got.ID])
	auth := iamEventDo(h.engine, http.MethodGet, "/admin/tenants", got.Secret, "")
	if auth.Code != http.StatusForbidden {
		t.Fatalf("viewer tenants status = %d, want 403, body = %s", auth.Code, auth.Body.String())
	}
}
