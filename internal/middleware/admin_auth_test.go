package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

type stubKeys struct {
	keys map[string]*models.ApiKey
}

func (s *stubKeys) FindActiveKeyByHash(keyHash string) (*models.ApiKey, error) {
	k := s.keys[keyHash]
	if k == nil || k.IsRevoked {
		return nil, nil
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	return k, nil
}

func (s *stubKeys) UpdateApiKeyLastUsed(id uuid.UUID) {}

func (s *stubKeys) IncrementDailyUsage(id uuid.UUID) {}

func hashRaw(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (s *stubKeys) put(raw string, key *models.ApiKey) {
	if s.keys == nil {
		s.keys = map[string]*models.ApiKey{}
	}
	key.KeyHash = hashRaw(raw)
	s.keys[key.KeyHash] = key
}

type stubGrants struct {
	byID map[uuid.UUID]struct {
		name string
		keys []string
	}
}

func (s *stubGrants) RoleGrants(_ context.Context, roleID uuid.UUID) (string, []string, error) {
	g, ok := s.byID[roleID]
	if !ok {
		return "", nil, pgx.ErrNoRows
	}
	return g.name, g.keys, nil
}

func adminGET(mw gin.HandlerFunc, extra gin.HandlerFunc, header string) *httptest.ResponseRecorder {
	return adminRequest(http.MethodGet, mw, extra, header, nil)
}

func adminRequest(method string, mw gin.HandlerFunc, extra gin.HandlerFunc, apiKey string, headers map[string]string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers := []gin.HandlerFunc{mw}
	if extra != nil {
		handlers = append(handlers, extra)
	}
	handlers = append(handlers, func(c *gin.Context) { c.Status(http.StatusOK) })
	r.Handle(method, "/admin/x", handlers...)
	req := httptest.NewRequest(method, "/admin/x", nil)
	if apiKey != "" {
		req.Header.Set("X-Admin-API-Key", apiKey)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAdminAuth_EnvKeyAttachesSuperadminAndSkipsScopes(t *testing.T) {
	const env = "env-secret"
	w := adminGET(AdminAuthMiddleware(env, nil, nil), func(c *gin.Context) {
		if _, ok := c.Get(web.ApiKeyScopesKey); ok {
			t.Error("env key must not set ApiKeyScopesKey")
		}
		p := mustPrincipal(t, c)
		if p.Kind != operator.KindEnvKey {
			t.Fatalf("kind = %q", p.Kind)
		}
		if !p.Has(operator.ResTenants, operator.ActionWrite) {
			t.Fatal("env principal must have tenants:write")
		}
		if !p.Has(operator.ResAdminIAM, operator.ActionWrite) {
			t.Fatal("env principal must have admin_iam:write")
		}
	}, env)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestAdminAuth_MissingHeader401(t *testing.T) {
	w := adminGET(AdminAuthMiddleware("env", nil, nil), nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAdminAuth_BadKey401(t *testing.T) {
	w := adminGET(AdminAuthMiddleware("env", nil, nil), nil, "nope")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAdminAuth_ExpiredDBKey401NoPrincipal(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	store := &stubKeys{}
	raw := "ak_admin_expired_key_value"
	store.put(raw, &models.ApiKey{
		ID:        uuid.New(),
		KeyType:   admin.KeyTypeAdmin,
		ExpiresAt: &past,
	})
	w := adminGET(AdminAuthMiddleware("", store, nil), func(c *gin.Context) {
		t.Fatal("handler must not run")
	}, raw)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestAdminAuth_ViewerDBKeyNoScopesKey(t *testing.T) {
	roleID := operator.RoleIDViewer
	store := &stubKeys{}
	raw := "ak_admin_viewer_key_value"
	id := uuid.New()
	store.put(raw, &models.ApiKey{
		ID:             id,
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &roleID,
	})
	grants := &stubGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		roleID: {name: operator.RoleViewer, keys: operator.GrantsFor(operator.RoleViewer)},
	}}
	w := adminGET(AdminAuthMiddleware("", store, grants), func(c *gin.Context) {
		if _, ok := c.Get(web.ApiKeyScopesKey); ok {
			t.Error("admin DB key must not set ApiKeyScopesKey")
		}
		p := mustPrincipal(t, c)
		if p.Kind != operator.KindAPIKey {
			t.Fatalf("kind = %q", p.Kind)
		}
		if p.RoleName != operator.RoleViewer {
			t.Fatalf("role = %q", p.RoleName)
		}
		if p.Has(operator.ResTenants, operator.ActionWrite) {
			t.Fatal("viewer must not have tenants:write")
		}
		if !p.Has(operator.ResLogs, operator.ActionRead) {
			t.Fatal("viewer must have logs:read")
		}
	}, raw)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestAdminAuth_NullRoleIsUnauthorized(t *testing.T) {
	store := &stubKeys{}
	raw := "ak_admin_legacy_key_value"
	store.put(raw, &models.ApiKey{
		ID:      uuid.New(),
		KeyType: admin.KeyTypeAdmin,
	})
	w := adminGET(AdminAuthMiddleware("", store, nil), func(c *gin.Context) {
		t.Fatal("handler must not run")
	}, raw)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Invalid Admin API Key") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestRequireOperatorPermission_EnvCanWriteTenants(t *testing.T) {
	const env = "env-secret"
	w := adminGET(
		AdminAuthMiddleware(env, nil, nil),
		RequireOperatorPermission(operator.ResTenants, operator.ActionWrite),
		env,
	)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}

func TestRequireOperatorPermission_ViewerCannotWriteTenants(t *testing.T) {
	roleID := operator.RoleIDViewer
	store := &stubKeys{}
	raw := "ak_admin_viewer2_key_value"
	store.put(raw, &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &roleID,
	})
	grants := &stubGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		roleID: {name: operator.RoleViewer, keys: operator.GrantsFor(operator.RoleViewer)},
	}}
	w := adminGET(
		AdminAuthMiddleware("", store, grants),
		RequireOperatorPermission(operator.ResTenants, operator.ActionWrite),
		raw,
	)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == "" {
		t.Fatal("expected error field")
	}
}

func TestRequireOperatorPermission_MissingPrincipal500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin/x", RequireOperatorPermission(operator.ResTenants, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestRequireOperatorPermission_BadType500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin/x", func(c *gin.Context) {
		c.Set(web.OperatorPrincipalKey, "nope")
		c.Next()
	}, RequireOperatorPermission(operator.ResTenants, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
}

func mustPrincipal(t *testing.T, c *gin.Context) *operator.Principal {
	t.Helper()
	val, ok := c.Get(web.OperatorPrincipalKey)
	if !ok {
		t.Fatal("missing principal")
	}
	p, ok := val.(*operator.Principal)
	if !ok || p == nil {
		t.Fatal("principal type")
	}
	return p
}
