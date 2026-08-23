package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/web"
)

func TestPage_ViewerNavOmitsTenants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/gui/users", nil)
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleViewer, operator.GrantsFor(operator.RoleViewer))
	c.Set(web.OperatorPrincipalKey, &p)
	c.Set(web.GUIAdminUsernameKey, "viewer")
	c.Set(web.GUIAdminIDKey, "id-1")
	c.Set(web.CSRFTokenKey, "csrf")

	h := &GUIHandler{BasePath: "/gui"}
	data := h.page(c)
	data.ActivePage = "users"

	if data.AdminUsername != "viewer" {
		t.Fatalf("username = %q", data.AdminUsername)
	}
	if !data.Can("users", "read") {
		t.Fatal("viewer cannot users:read")
	}
	if data.Can("tenants", "read") {
		t.Fatal("viewer can tenants:read")
	}
	for _, g := range data.NavGroups {
		for _, item := range g.Items {
			if item.Label == "Tenants" {
				t.Fatal("viewer NavGroups includes Tenants")
			}
		}
	}
}

func TestPage_BoundAdminNavOmitsPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/gui/users", nil)
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleAdmin, operator.GrantsFor(operator.RoleAdmin))
	appID := uuid.New()
	p.AppID = &appID
	c.Set(web.OperatorPrincipalKey, &p)
	c.Set(web.GUIAdminUsernameKey, "bound-admin")
	c.Set(web.GUIAdminIDKey, "id-bound")
	c.Set(web.CSRFTokenKey, "csrf")

	h := &GUIHandler{BasePath: "/gui"}
	data := h.page(c)
	if !data.Can("users", "read") {
		t.Fatal("bound admin cannot users:read")
	}
	if data.Can("tenants", "read") {
		t.Fatal("bound admin can tenants:read")
	}
	if data.Can("dashboard", "read") {
		t.Fatal("bound admin can dashboard:read")
	}
	forbidden := map[string]struct{}{
		"Tenants":        {},
		"Operator IAM":   {},
		"Dashboard":      {},
		"Settings":       {},
		"Applications":   {},
		"System Health":  {},
		"Email Servers":  {},
		"OIDC Clients":   {},
		"Session Groups": {},
	}
	for _, g := range data.NavGroups {
		if g.Heading == "Email" {
			t.Fatal("bound admin nav has Email heading")
		}
		for _, item := range g.Items {
			if _, bad := forbidden[item.Label]; bad {
				t.Fatalf("bound admin nav includes %s", item.Label)
			}
		}
	}
}

func TestPage_MissingPrincipalDenies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/gui/", nil)
	h := &GUIHandler{BasePath: "/gui"}
	data := h.page(c)
	if data.Can("dashboard", "read") {
		t.Fatal("nil principal allowed dashboard:read")
	}
	if len(data.NavGroups) != 0 {
		t.Fatalf("NavGroups = %#v", data.NavGroups)
	}
}
