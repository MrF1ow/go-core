package coreapp

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
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

const (
	accessEnvKey        = "access-env-key"
	accessViewerKey     = "access-viewer-key"
	accessSuperadminKey = "access-superadmin-key"
	accessAdminKey      = "access-admin-key"
	accessSupportKey    = "access-support-key"
)

type accessLogKeyStore struct {
	keys map[string]*models.ApiKey
}

func (s *accessLogKeyStore) FindActiveKeyByHash(keyHash string) (*models.ApiKey, error) {
	return s.keys[keyHash], nil
}

func (s *accessLogKeyStore) UpdateApiKeyLastUsed(uuid.UUID) {}

func (s *accessLogKeyStore) IncrementDailyUsage(uuid.UUID) {}

func (s *accessLogKeyStore) put(raw string, key *models.ApiKey) {
	if s.keys == nil {
		s.keys = map[string]*models.ApiKey{}
	}
	sum := sha256.Sum256([]byte(raw))
	key.KeyHash = hex.EncodeToString(sum[:])
	s.keys[key.KeyHash] = key
}

type accessLogGrants struct {
	byID map[uuid.UUID]struct {
		name string
		keys []string
	}
}

func (s *accessLogGrants) RoleGrants(_ context.Context, roleID uuid.UUID) (string, []string, error) {
	g, ok := s.byID[roleID]
	if !ok {
		return "", nil, pgx.ErrNoRows
	}
	return g.name, g.keys, nil
}

type accessLogMem struct {
	mu      sync.Mutex
	records []operator.AccessRecord
}

func (m *accessLogMem) append(rec operator.AccessRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, rec)
}

func (m *accessLogMem) list(_ context.Context, limit int32, decision *string) ([]operator.AccessRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]operator.AccessRecord, 0, len(m.records))
	for _, rec := range m.records {
		if decision != nil && rec.Decision != *decision {
			continue
		}
		out = append(out, rec)
		if int32(len(out)) >= limit {
			break
		}
	}
	return out, nil
}

func accessLogTestEngine(t *testing.T) (*gin.Engine, *accessLogMem) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store := &accessLogKeyStore{}
	viewerRole := operator.RoleIDViewer
	superRole := operator.RoleIDSuperadmin
	store.put(accessViewerKey, &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &viewerRole,
	})
	store.put(accessSuperadminKey, &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &superRole,
	})
	grants := &accessLogGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		viewerRole: {name: operator.RoleViewer, keys: operator.GrantsFor(operator.RoleViewer)},
		superRole:  {name: operator.RoleSuperadmin, keys: operator.GrantsFor(operator.RoleSuperadmin)},
	}}

	mem := &accessLogMem{}
	middleware.SetOperatorAccessLogger(mem.append)
	t.Cleanup(func() { middleware.SetOperatorAccessLogger(nil) })

	handler := &admin.Handler{AccessLogList: mem.list}
	engine := gin.New()
	group := engine.Group("/admin")
	group.Use(middleware.AdminAuthMiddleware(accessEnvKey, store, grants))
	group.POST("/tenants", requireOp(operator.ResTenants, operator.ActionWrite), func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})
	group.GET("/activity-logs", requireOp(operator.ResLogs, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	group.GET("/operator/access-logs", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorAccessLogs)
	group.GET("/operator/access-logs/export", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorAccessLogsExport)
	return engine, mem
}

func accessLogDo(engine *gin.Engine, method, path, key string) *httptest.ResponseRecorder {
	return accessLogDoHeaders(engine, method, path, key, nil)
}

func accessLogDoHeaders(engine *gin.Engine, method, path, key string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Admin-API-Key", key)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func accessLogList(t *testing.T, engine *gin.Engine) []operator.AccessRecord {
	t.Helper()
	rec := accessLogDo(engine, http.MethodGet, "/admin/operator/access-logs", accessSuperadminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries []operator.AccessRecord `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Entries
}

func TestOperatorAccessLog_ViewerTenantsWriteIsDeny(t *testing.T) {
	engine, _ := accessLogTestEngine(t)
	rec := accessLogDo(engine, http.MethodPost, "/admin/tenants", accessViewerKey)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, entry := range accessLogList(t, engine) {
		if entry.Path == "/admin/tenants" && entry.Decision == operator.DecisionDeny {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected deny entry for /admin/tenants")
	}
}

func TestOperatorAccessLog_EnvTenantsWriteIsAllow(t *testing.T) {
	engine, _ := accessLogTestEngine(t)
	rec := accessLogDo(engine, http.MethodPost, "/admin/tenants", accessEnvKey)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, entry := range accessLogList(t, engine) {
		if entry.Path == "/admin/tenants" && entry.Decision == operator.DecisionAllow && entry.Kind == operator.KindEnvKey {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected allow env_key entry for /admin/tenants")
	}
}

func TestOperatorAccessLog_ViewerActivityLogsReadNotLogged(t *testing.T) {
	engine, _ := accessLogTestEngine(t)
	rec := accessLogDo(engine, http.MethodGet, "/admin/activity-logs", accessViewerKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	for _, entry := range accessLogList(t, engine) {
		if entry.Path == "/admin/activity-logs" && entry.Decision == operator.DecisionAllow {
			t.Fatalf("ordinary read allow must not log: %+v", entry)
		}
	}
}

func TestOperatorAccessLog_ViewerListForbidden(t *testing.T) {
	engine, _ := accessLogTestEngine(t)
	rec := accessLogDo(engine, http.MethodGet, "/admin/operator/access-logs", accessViewerKey)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestOperatorAccessLogExport_ViewerForbiddenJSON(t *testing.T) {
	engine, _ := accessLogTestEngine(t)
	rec := accessLogDo(engine, http.MethodGet, "/admin/operator/access-logs/export", accessViewerKey)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestOperatorAccessLog_ViewerTenantsWriteCapturesClient(t *testing.T) {
	engine, _ := accessLogTestEngine(t)
	rec := accessLogDoHeaders(engine, http.MethodPost, "/admin/tenants", accessViewerKey, map[string]string{
		"X-Forwarded-For": "203.0.113.9",
		"User-Agent":      "AccessClientTest/1.0",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, entry := range accessLogList(t, engine) {
		if entry.Path == "/admin/tenants" && entry.Decision == operator.DecisionDeny {
			if entry.IPAddress != "203.0.113.9" {
				t.Fatalf("ip = %q", entry.IPAddress)
			}
			if entry.UserAgent != "AccessClientTest/1.0" {
				t.Fatalf("ua = %q", entry.UserAgent)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected deny entry for /admin/tenants")
	}
}

func TestOperatorAccessLogExport_SuperadminCSV(t *testing.T) {
	engine, _ := accessLogTestEngine(t)
	deny := accessLogDoHeaders(engine, http.MethodPost, "/admin/tenants", accessViewerKey, map[string]string{
		"X-Forwarded-For": "203.0.113.9",
		"User-Agent":      "AccessClientTest/1.0",
	})
	if deny.Code != http.StatusForbidden {
		t.Fatalf("deny status = %d", deny.Code)
	}
	rec := accessLogDo(engine, http.MethodGet, "/admin/operator/access-logs/export", accessSuperadminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Export-Truncated") != "false" {
		t.Fatalf("truncated = %q", rec.Header().Get("X-Export-Truncated"))
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "operator-access-logs.csv") {
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
	if len(rows) < 2 {
		t.Fatalf("csv rows = %#v", rows)
	}
	wantHeader := []string{"id", "at", "kind", "key_id", "account_id", "role_name", "method", "path", "decision", "resource", "action", "status", "ip_address", "user_agent"}
	if strings.Join(rows[0], ",") != strings.Join(wantHeader, ",") {
		t.Fatalf("header = %#v", rows[0])
	}
	found := false
	for _, row := range rows[1:] {
		if len(row) != len(wantHeader) {
			t.Fatalf("row width = %d, want %d: %#v", len(row), len(wantHeader), row)
		}
		if row[7] == "/admin/tenants" && row[8] == operator.DecisionDeny {
			if row[12] != "203.0.113.9" {
				t.Fatalf("csv ip = %q", row[12])
			}
			if row[13] != "AccessClientTest/1.0" {
				t.Fatalf("csv ua = %q", row[13])
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("missing deny tenants row in %#v", rows)
	}
}

func TestOperatorAccessLogExport_TruncatesAtExportMaxRows(t *testing.T) {
	engine, mem := accessLogTestEngine(t)
	for i := 0; i < operator.ExportMaxRows+1; i++ {
		mem.append(operator.AccessRecord{Decision: operator.DecisionDeny, Path: "/admin/tenants"})
	}
	rec := accessLogDo(engine, http.MethodGet, "/admin/operator/access-logs/export", accessSuperadminKey)
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

func TestOperatorAccessLogs_ListLimitStillCapsAt1000(t *testing.T) {
	engine, mem := accessLogTestEngine(t)
	for i := 0; i < 1500; i++ {
		mem.append(operator.AccessRecord{Decision: operator.DecisionDeny, Path: "/admin/tenants"})
	}
	rec := accessLogDo(engine, http.MethodGet, "/admin/operator/access-logs?limit=5000", accessSuperadminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries []operator.AccessRecord `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != 1000 {
		t.Fatalf("len = %d, want 1000", len(body.Entries))
	}
}
