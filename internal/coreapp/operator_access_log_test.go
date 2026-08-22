package coreapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	return engine, mem
}

func accessLogDo(engine *gin.Engine, method, path, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Admin-API-Key", key)
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
