package coreapp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

type stubGUISessions struct {
	account *models.AdminAccount
}

func (s *stubGUISessions) ValidateSession(string) (*models.AdminAccount, error) {
	return s.account, nil
}

func (s *stubGUISessions) GenerateCSRFToken(string) (string, error) {
	return "csrf-token", nil
}

func (s *stubGUISessions) ValidateCSRFToken(string, string) bool {
	return true
}

type stubGUIGrants struct {
	byID map[uuid.UUID]struct {
		name string
		keys []string
	}
}

func (s *stubGUIGrants) RoleGrants(_ context.Context, roleID uuid.UUID) (string, []string, error) {
	grants, ok := s.byID[roleID]
	if !ok {
		return "", nil, pgx.ErrNoRows
	}
	return grants.name, grants.keys, nil
}

func TestGUIShell_ViewerTenantsForbidden(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/tenants", cookie, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
	body := response.Body.String()
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("JSON body = %s", body)
	}
}

func TestGUIShell_SuperadminTenantsOK(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(engine, "/gui/tenants", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGUIShell_ViewerSidebarOmitsTenantsAndEmptyEmail(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/users", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "Users") {
		t.Fatal("viewer sidebar missing Users")
	}
	if !strings.Contains(body, "Activity Logs") {
		t.Fatal("viewer sidebar missing Activity Logs")
	}
	if !strings.Contains(body, "System Health") {
		t.Fatal("viewer sidebar missing System Health")
	}
	if strings.Contains(body, ">Tenants<") {
		t.Fatal("viewer sidebar includes Tenants")
	}
	if strings.Contains(body, `sidebar-heading">Email`) {
		t.Fatal("viewer sidebar has empty Email heading")
	}
	if strings.Contains(body, "Create Tenant") {
		t.Fatal("viewer saw Create Tenant")
	}
}

func TestGUIShell_ViewerUsersOmitsImport(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/users", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, ">Import<") || strings.Contains(body, "Import</button>") {
		t.Fatal("viewer saw Import button")
	}
}

func TestGUIShell_SupportNestedSessionsOK(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDSupport, operator.RoleSupport)
	response := guiGET(engine, "/gui/users/"+uuid.New().String()+"/sessions", cookie, "true", "user-sessions-container")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGUIShell_ViewerNestedSessionsForbidden(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/users/"+uuid.New().String()+"/sessions", cookie, "true", "user-sessions-container")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
	if !strings.Contains(response.Body.String(), "You do not have permission") {
		t.Fatalf("body = %s", response.Body.String())
	}
	if strings.Contains(response.Body.String(), `id="user-sessions-container"`) {
		t.Fatalf("fragment repeated target id: %s", response.Body.String())
	}
}

func TestGUIShell_ViewerUserDetailOK(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/users/"+uuid.New().String(), cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestGUIShell_ViewerLogoutNotForbidden(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/logout", cookie, "", "")
	if response.Code == http.StatusForbidden {
		t.Fatalf("logout was 403: %s", response.Body.String())
	}
}

func TestGUIShell_ViewerMyAccountOK(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/my-account", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGUIShell_AdminCannotStampSuperadminKey(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDAdmin, operator.RoleAdmin)
	form := url.Values{
		"name":             {"ops"},
		"key_type":         {admin.KeyTypeAdmin},
		"operator_role_id": {operator.RoleIDSuperadmin.String()},
	}
	response := guiPOST(engine, "/gui/api-keys", cookie, form)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
}

func TestGUIShell_AdminOmittingRoleIsNotIAMDeny(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDAdmin, operator.RoleAdmin)
	form := url.Values{
		"name":     {"ops"},
		"key_type": {admin.KeyTypeAdmin},
	}
	response := guiPOST(engine, "/gui/api-keys", cookie, form)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Body.String() != operator.RoleIDViewer.String() {
		t.Fatalf("role = %q, want viewer", response.Body.String())
	}
}

func TestGUIShell_AdminApiKeyFormOmitsRoleSelect(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDAdmin, operator.RoleAdmin)
	response := guiGET(engine, "/gui/api-keys/new", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, `name="operator_role_id"`) {
		t.Fatal("admin saw operator role field")
	}
}

func TestGUIShell_SuperadminApiKeyFormHasRoleSelect(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(engine, "/gui/api-keys/new", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `name="operator_role_id"`) {
		t.Fatal("superadmin missing operator role select")
	}
	if !strings.Contains(body, operator.RoleIDSuperadmin.String()) {
		t.Fatal("superadmin select missing superadmin option")
	}
}

func TestGUIShell_SuperadminSidebarHasTenants(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(engine, "/gui/", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, ">Tenants<") {
		t.Fatal("superadmin sidebar missing Tenants")
	}
	if !strings.Contains(body, "Settings") {
		t.Fatal("superadmin sidebar missing Settings")
	}
	if !strings.Contains(body, "API Keys") {
		t.Fatal("superadmin sidebar missing API Keys")
	}
}

func guiShellEngine(t *testing.T, roleID uuid.UUID, roleName string) (*gin.Engine, *http.Cookie) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	account := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       roleName,
		OperatorRoleID: roleID,
	}
	sessions := &stubGUISessions{account: account}
	grants := &stubGUIGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		roleID: {name: roleName, keys: operator.GrantsFor(roleName)},
	}}
	h := &admin.GUIHandler{
		BasePath:       "/gui",
		AbortForbidden: middleware.AbortGUIForbidden,
		AbortInternal:  middleware.AbortGUIInternal,
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.HTMLRender = renderer
	gui := router.Group("/gui")
	guiAuth := gui.Group("/")
	guiAuth.Use(middleware.GUIAuthMiddleware(sessions, grants, "/gui"))
	guiAuth.Use(middleware.CSRFMiddleware(sessions))
	guiAuth.GET("/", requireGUI(operator.ResDashboard, operator.ActionRead), h.Dashboard)
	guiAuth.GET("/tenants", requireGUI(operator.ResTenants, operator.ActionRead), h.TenantPage)
	guiAuth.GET("/users", requireGUI(operator.ResUsers, operator.ActionRead), func(c *gin.Context) {
		data := shellPage(c)
		data.ActivePage = "users"
		c.HTML(http.StatusOK, "users", data)
	})
	guiAuth.GET("/users/:id", requireGUI(operator.ResUsers, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	guiAuth.GET("/users/:id/sessions", requireGUI(operator.ResSessions, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	guiAuth.GET("/logout", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/gui/login")
	})
	guiAuth.GET("/my-account", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	guiAuth.GET("/api-keys/new", requireGUI(operator.ResAPIKeys, operator.ActionWrite), func(c *gin.Context) {
		canIAM := false
		if val, ok := c.Get(web.OperatorPrincipalKey); ok {
			if p, ok := val.(*operator.Principal); ok && p != nil {
				canIAM = p.Has(operator.ResAdminIAM, operator.ActionWrite)
			}
		}
		c.HTML(http.StatusOK, "api_key_form", gin.H{
			"OperatorRoles": []struct {
				ID   uuid.UUID
				Name string
			}{
				{ID: operator.RoleIDViewer, Name: operator.RoleViewer},
				{ID: operator.RoleIDSupport, Name: operator.RoleSupport},
				{ID: operator.RoleIDAdmin, Name: operator.RoleAdmin},
				{ID: operator.RoleIDSuperadmin, Name: operator.RoleSuperadmin},
			},
			"DefaultRoleID": operator.RoleIDViewer.String(),
			"CanIAM":        canIAM,
		})
	})
	guiAuth.POST("/api-keys", requireGUI(operator.ResAPIKeys, operator.ActionWrite), func(c *gin.Context) {
		val, ok := c.Get(web.OperatorPrincipalKey)
		if !ok {
			middleware.AbortGUIInternal(c)
			return
		}
		p, ok := val.(*operator.Principal)
		if !ok || p == nil {
			middleware.AbortGUIInternal(c)
			return
		}
		roleID, err := operator.ParseAssignedAdminRole(*p, c.PostForm("operator_role_id"), c.PostForm("key_type"), nil)
		if errors.Is(err, operator.ErrIAMAssignmentDenied) {
			middleware.AbortGUIForbidden(c)
			return
		}
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if roleID == nil {
			c.Status(http.StatusOK)
			return
		}
		c.String(http.StatusOK, roleID.String())
	})
	return router, &http.Cookie{Name: web.AdminSessionCookie, Value: "session-id"}
}

func shellPage(c *gin.Context) web.TemplateData {
	data := web.TemplateData{
		Theme:         web.GetTheme(c),
		AdminUsername: c.GetString(web.GUIAdminUsernameKey),
		AdminID:       c.GetString(web.GUIAdminIDKey),
		CSRFToken:     c.GetString(web.CSRFTokenKey),
	}
	val, ok := c.Get(web.OperatorPrincipalKey)
	if !ok {
		return data
	}
	p, ok := val.(*operator.Principal)
	if !ok || p == nil {
		return data
	}
	return web.AttachCan(data, "/gui", p.Has)
}

func guiGET(engine *gin.Engine, path string, cookie *http.Cookie, hxRequest, hxTarget string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.AddCookie(cookie)
	if hxRequest != "" {
		request.Header.Set("HX-Request", hxRequest)
		request.Header.Set("HX-Target", hxTarget)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func guiPOST(engine *gin.Engine, path string, cookie *http.Cookie, form url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	request.AddCookie(cookie)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRF-Token", "csrf-token")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}
