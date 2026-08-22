package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

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
