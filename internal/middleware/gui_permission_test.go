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
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
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

func TestRequireGUIPermission_HTMXFragmentDoesNotRepeatTargetID(t *testing.T) {
	response := guiPermissionGET(t, viewerPrincipal(), operator.ResSessions, operator.ActionRead, "user-sessions-container")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "You do not have permission") {
		t.Fatalf("body = %s", body)
	}
	if strings.Contains(body, `id="user-sessions-container"`) {
		t.Fatalf("fragment repeated target id: %s", body)
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
	if strings.Contains(body, `"><script>alert(1)</script>`) {
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

const csrfForbiddenCopy = "Reload this page and submit the form again."

type rejectCSRFSessions struct {
	stubGUISessions
}

func (s *rejectCSRFSessions) ValidateCSRFToken(string, string) bool {
	return false
}

func csrfForbiddenPOST(t *testing.T, validator web.SessionValidator, token string, hxTarget string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.Use(func(c *gin.Context) {
		c.Set(web.GUISessionIDKey, "sess")
		c.Set(web.GUIAdminBasePathKey, "/gui")
		c.Set(web.GUIAdminUsernameKey, "tester")
		c.Next()
	})
	router.Use(CSRFMiddleware(validator))
	router.POST("/gui/tenants", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodPost, "/gui/tenants", nil)
	if token != "" {
		request.Header.Set("X-CSRF-Token", token)
	}
	if hxTarget != "" {
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", hxTarget)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestCSRFForbiddenDoesNotSendGUIHeader(t *testing.T) {
	response := csrfForbiddenPOST(t, &stubGUISessions{}, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("CSRF 403 sent %s", web.GUIForbiddenHeader)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	body := response.Body.String()
	if strings.Contains(body, `{"error":"CSRF token missing"}`) {
		t.Fatalf("JSON body = %s", body)
	}
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("JSON body = %s", body)
	}
	if !strings.Contains(body, csrfForbiddenCopy) {
		t.Fatalf("body = %s", body)
	}
	if strings.Contains(body, "You do not have permission") {
		t.Fatalf("reused IAM copy: %s", body)
	}
	if !strings.Contains(body, `id="page-content"`) {
		t.Fatalf("typed POST missing page template: %s", body)
	}
}

func TestCSRFForbiddenHTMXIsFragmentWithoutGUIHeader(t *testing.T) {
	response := csrfForbiddenPOST(t, &stubGUISessions{}, "", "tenant-form-container")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("CSRF 403 sent %s", web.GUIForbiddenHeader)
	}
	body := response.Body.String()
	if !strings.Contains(body, csrfForbiddenCopy) {
		t.Fatalf("body = %s", body)
	}
	if strings.Contains(body, "<nav") {
		t.Fatalf("HTMX CSRF leaked layout: %s", body)
	}
	if strings.Contains(body, `id="tenant-form-container"`) {
		t.Fatalf("fragment repeated target id: %s", body)
	}
}

func TestCSRFInvalidTokenIsHTMLWithoutGUIHeader(t *testing.T) {
	response := csrfForbiddenPOST(t, &rejectCSRFSessions{}, "nope", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("CSRF 403 sent %s", web.GUIForbiddenHeader)
	}
	if strings.Contains(response.Body.String(), `{"error":"CSRF token invalid"}`) {
		t.Fatalf("JSON body = %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), csrfForbiddenCopy) {
		t.Fatalf("body = %s", response.Body.String())
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
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.Use(func(c *gin.Context) {
		c.Set(web.OperatorPrincipalKey, &principal)
		c.Set(web.GUIAdminBasePathKey, "/gui")
		c.Set(web.GUIAdminUsernameKey, "tester")
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
