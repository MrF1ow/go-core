package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

func TestApiKeyCreate_AdminCannotStampSuperadmin(t *testing.T) {
	got := apiKeyCreatePOST(t, adminPrincipal(), url.Values{
		"name":             {"ops"},
		"key_type":         {KeyTypeAdmin},
		"operator_role_id": {operator.RoleIDSuperadmin.String()},
	})
	if got.response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", got.response.Code, got.response.Body.String())
	}
	if got.response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
	if got.created != nil {
		t.Fatal("denied stamp still persisted a key")
	}
}

func TestApiKeyCreate_AdminOmittingRoleIsNotIAMDeny(t *testing.T) {
	got := apiKeyCreatePOST(t, adminPrincipal(), url.Values{
		"name":     {"ops"},
		"key_type": {KeyTypeAdmin},
	})
	if got.response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got.response.Code, got.response.Body.String())
	}
	if got.response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("sent %s", web.GUIForbiddenHeader)
	}
	assertPersistedRole(t, got.created, operator.RoleIDViewer)
}

func TestApiKeyCreate_SuperadminCanStampSuperadminPastIAM(t *testing.T) {
	got := apiKeyCreatePOST(t, superadminGUIPrincipal(), url.Values{
		"name":             {"root-key"},
		"key_type":         {KeyTypeAdmin},
		"operator_role_id": {operator.RoleIDSuperadmin.String()},
	})
	if got.response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got.response.Code, got.response.Body.String())
	}
	if got.response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("sent %s", web.GUIForbiddenHeader)
	}
	assertPersistedRole(t, got.created, operator.RoleIDSuperadmin)
}

func adminPrincipal() operator.Principal {
	return operator.NewPrincipal(operator.KindGUIAccount, operator.RoleAdmin, operator.GrantsFor(operator.RoleAdmin))
}

func superadminGUIPrincipal() operator.Principal {
	return operator.NewPrincipal(operator.KindGUIAccount, operator.RoleSuperadmin, operator.GrantsFor(operator.RoleSuperadmin))
}

func assertPersistedRole(t *testing.T, created *models.ApiKey, want uuid.UUID) {
	t.Helper()
	if created == nil {
		t.Fatal("no key persisted")
	}
	if created.OperatorRoleID == nil || *created.OperatorRoleID != want {
		t.Fatalf("role = %v, want %s", created.OperatorRoleID, want)
	}
}

type apiKeyCreateResult struct {
	response *httptest.ResponseRecorder
	created  *models.ApiKey
}

func apiKeyCreatePOST(t *testing.T, principal operator.Principal, form url.Values) apiKeyCreateResult {
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
		c.Next()
	})
	var created *models.ApiKey
	h := &GUIHandler{
		BasePath: "/gui",
		AbortForbidden: func(c *gin.Context) {
			c.Header(web.GUIForbiddenHeader, web.GUIForbiddenValue)
			c.AbortWithStatus(http.StatusForbidden)
		},
		AbortInternal: func(c *gin.Context) {
			c.AbortWithStatus(http.StatusInternalServerError)
		},
		createAPIKey: func(key *models.ApiKey) error {
			stored := *key
			created = &stored
			return nil
		},
	}
	router.POST("/gui/api-keys", h.ApiKeyCreate)

	request := httptest.NewRequest(http.MethodPost, "/gui/api-keys", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return apiKeyCreateResult{response: response, created: created}
}
