package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
)

func captureAccessLogs(t *testing.T) *[]operator.AccessRecord {
	t.Helper()
	recs := &[]operator.AccessRecord{}
	SetOperatorAccessLogger(func(rec operator.AccessRecord) {
		*recs = append(*recs, rec)
	})
	t.Cleanup(func() { SetOperatorAccessLogger(nil) })
	return recs
}

func viewerAdminStore(t *testing.T) (raw string, store *stubKeys, grants *stubGrants) {
	t.Helper()
	roleID := operator.RoleIDViewer
	store = &stubKeys{}
	raw = "ak_admin_access_log_viewer"
	store.put(raw, &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &roleID,
	})
	grants = &stubGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		roleID: {name: operator.RoleViewer, keys: operator.GrantsFor(operator.RoleViewer)},
	}}
	return raw, store, grants
}

func TestRequireOperatorPermission_ViewerWriteLogsDeny(t *testing.T) {
	recs := captureAccessLogs(t)
	raw, store, grants := viewerAdminStore(t)
	w := adminGET(
		AdminAuthMiddleware("", store, grants),
		RequireOperatorPermission(operator.ResTenants, operator.ActionWrite),
		raw,
	)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if len(*recs) != 1 {
		t.Fatalf("logged %d records, want 1: %#v", len(*recs), *recs)
	}
	got := (*recs)[0]
	if got.Decision != operator.DecisionDeny {
		t.Fatalf("decision = %q", got.Decision)
	}
	if got.Resource != operator.ResTenants || got.Action != operator.ActionWrite {
		t.Fatalf("grant = %s:%s", got.Resource, got.Action)
	}
	if got.Path != "/admin/x" {
		t.Fatalf("path = %q", got.Path)
	}
	if got.Status != http.StatusForbidden {
		t.Fatalf("status = %d", got.Status)
	}
}

func TestRequireOperatorPermission_EnvWriteLogsAllow(t *testing.T) {
	recs := captureAccessLogs(t)
	const env = "env-secret"
	w := adminGET(
		AdminAuthMiddleware(env, nil, nil),
		RequireOperatorPermission(operator.ResTenants, operator.ActionWrite),
		env,
	)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if len(*recs) != 1 {
		t.Fatalf("logged %d records, want 1: %#v", len(*recs), *recs)
	}
	got := (*recs)[0]
	if got.Decision != operator.DecisionAllow {
		t.Fatalf("decision = %q", got.Decision)
	}
	if got.Kind != operator.KindEnvKey {
		t.Fatalf("kind = %q", got.Kind)
	}
	if got.Status != http.StatusOK {
		t.Fatalf("status = %d", got.Status)
	}
}

func TestRequireOperatorPermission_ViewerReadAllowLogsNothing(t *testing.T) {
	recs := captureAccessLogs(t)
	raw, store, grants := viewerAdminStore(t)
	w := adminGET(
		AdminAuthMiddleware("", store, grants),
		RequireOperatorPermission(operator.ResLogs, operator.ActionRead),
		raw,
	)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if len(*recs) != 0 {
		t.Fatalf("ordinary read allow must not log: %#v", *recs)
	}
}

func TestRequireOperatorPermission_MissingPrincipalLogsNothing(t *testing.T) {
	recs := captureAccessLogs(t)
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
	if len(*recs) != 0 {
		t.Fatalf("missing principal must not log: %#v", *recs)
	}
}

func TestRequireGUIPermission_ViewerWriteLogsDeny(t *testing.T) {
	recs := captureAccessLogs(t)
	response := guiPermissionGET(t, viewerPrincipal(), operator.ResTenants, operator.ActionWrite, "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(*recs) != 1 {
		t.Fatalf("logged %d records, want 1: %#v", len(*recs), *recs)
	}
	got := (*recs)[0]
	if got.Decision != operator.DecisionDeny {
		t.Fatalf("decision = %q", got.Decision)
	}
	if got.Kind != operator.KindGUIAccount {
		t.Fatalf("kind = %q", got.Kind)
	}
	if got.Status != http.StatusForbidden {
		t.Fatalf("status = %d", got.Status)
	}
}

func TestRequireGUIPermission_MissingPrincipalLogsNothing(t *testing.T) {
	recs := captureAccessLogs(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/gui/tenants", RequireGUIPermission(operator.ResTenants, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/gui/tenants", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", response.Code)
	}
	if len(*recs) != 0 {
		t.Fatalf("missing GUI principal must not log: %#v", *recs)
	}
}

func TestRequireOperatorPermission_NilLoggerStillAllows(t *testing.T) {
	SetOperatorAccessLogger(nil)
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
