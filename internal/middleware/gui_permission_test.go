package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/web"
)

func TestRequireGUIPermission_ViewerDeniedTenantsWriteIsHTML(t *testing.T) {
	response := guiPermissionGET(t, viewerPrincipal(), operator.ResTenants, operator.ActionWrite, "")

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("forbidden header = %q", response.Header().Get(web.GUIForbiddenHeader))
	}
	body := response.Body.String()
	if strings.Contains(body, "{") || strings.Contains(body, `"error"`) {
		t.Fatalf("JSON body = %s", body)
	}
	if !strings.Contains(body, `id="page-content"`) {
		t.Fatalf("missing page-content: %s", body)
	}
}

func TestRequireGUIPermission_SuperadminAllowed(t *testing.T) {
	response := guiPermissionGET(t, superadminPrincipal(), operator.ResTenants, operator.ActionWrite, "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRequireGUIPermission_HTMXPageTargetContainsPageContent(t *testing.T) {
	response := guiPermissionGET(t, viewerPrincipal(), operator.ResTenants, operator.ActionWrite, "page-content")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `id="page-content"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestRequireGUIPermission_HTMXFragmentTargetMatchesID(t *testing.T) {
	response := guiPermissionGET(t, viewerPrincipal(), operator.ResSessions, operator.ActionRead, "user-sessions-container")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `id="user-sessions-container"`) {
		t.Fatalf("body = %s", body)
	}
	if strings.Contains(body, "<nav") {
		t.Fatalf("fragment leaked layout: %s", body)
	}
}

func TestRequireGUIPermission_CraftedTargetFallsBackToPage(t *testing.T) {
	response := guiPermissionGET(t, viewerPrincipal(), operator.ResTenants, operator.ActionWrite, `"><script>alert(1)</script>`)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatalf("unsanitized target in body: %s", body)
	}
	if !strings.Contains(body, `id="page-content"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestRequireGUIPermission_MissingPrincipalIs500HTML(t *testing.T) {
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
	if strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if strings.Contains(response.Body.String(), "{") {
		t.Fatalf("JSON body = %s", response.Body.String())
	}
}

func TestRequireOperatorPermission_StillJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	p := operator.NewPrincipal(operator.KindAPIKey, operator.RoleViewer, operator.GrantsFor(operator.RoleViewer))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(web.OperatorPrincipalKey, &p)
		c.Next()
	})
	router.GET("/admin/tenants", RequireOperatorPermission(operator.ResTenants, operator.ActionWrite), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/admin/tenants", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
}

func viewerPrincipal() operator.Principal {
	return operator.NewPrincipal(operator.KindGUIAccount, operator.RoleViewer, operator.GrantsFor(operator.RoleViewer))
}

func superadminPrincipal() operator.Principal {
	return operator.NewPrincipal(operator.KindGUIAccount, operator.RoleSuperadmin, operator.GrantsFor(operator.RoleSuperadmin))
}

func guiPermissionGET(t *testing.T, principal operator.Principal, resource, action, hxTarget string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(web.OperatorPrincipalKey, &principal)
		c.Next()
	})
	router.GET("/gui/check", RequireGUIPermission(resource, action), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/gui/check", nil)
	if hxTarget != "" {
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", hxTarget)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
