package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/web"
)

func TestApiKeyCreate_AdminCannotStampSuperadmin(t *testing.T) {
	response := apiKeyCreatePOST(t, adminPrincipal(), url.Values{
		"name":             {"ops"},
		"key_type":         {KeyTypeAdmin},
		"operator_role_id": {operator.RoleIDSuperadmin.String()},
	})
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
}

func TestApiKeyCreate_AdminOmittingRoleIsNotIAMDeny(t *testing.T) {
	response := apiKeyCreatePOST(t, adminPrincipal(), url.Values{
		"name":     {"ops"},
		"key_type": {KeyTypeAdmin},
	})
	if response.Code == http.StatusForbidden {
		t.Fatalf("empty role was treated as IAM deny: %s", response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("sent %s", web.GUIForbiddenHeader)
	}
}

func TestApiKeyCreate_SuperadminCanStampSuperadminPastIAM(t *testing.T) {
	response := apiKeyCreatePOST(t, superadminGUIPrincipal(), url.Values{
		"name":             {"root-key"},
		"key_type":         {KeyTypeAdmin},
		"operator_role_id": {operator.RoleIDSuperadmin.String()},
	})
	if response.Code == http.StatusForbidden {
		t.Fatalf("superadmin IAM denied: %s", response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("sent %s", web.GUIForbiddenHeader)
	}
}

func adminPrincipal() operator.Principal {
	return operator.NewPrincipal(operator.KindGUIAccount, operator.RoleAdmin, operator.GrantsFor(operator.RoleAdmin))
}

func superadminGUIPrincipal() operator.Principal {
	return operator.NewPrincipal(operator.KindGUIAccount, operator.RoleSuperadmin, operator.GrantsFor(operator.RoleSuperadmin))
}

func apiKeyCreatePOST(t *testing.T, principal operator.Principal, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.HTMLRender = renderer
	router.Use(func(c *gin.Context) {
		c.Set(web.OperatorPrincipalKey, &principal)
		c.Set(web.GUIAdminBasePathKey, "/gui")
		c.Next()
	})
	h := &GUIHandler{
		BasePath: "/gui",
		AbortForbidden: func(c *gin.Context) {
			c.Header(web.GUIForbiddenHeader, web.GUIForbiddenValue)
			c.AbortWithStatus(http.StatusForbidden)
		},
		AbortInternal: func(c *gin.Context) {
			c.AbortWithStatus(http.StatusInternalServerError)
		},
	}
	router.POST("/gui/api-keys", h.ApiKeyCreate)

	request := httptest.NewRequest(http.MethodPost, "/gui/api-keys", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
