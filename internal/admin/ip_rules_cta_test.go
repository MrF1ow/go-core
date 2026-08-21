package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/web"
)

func TestIPRulesPage_ReadOnlyOmitsWriteControls(t *testing.T) {
	body := renderIPRulesPage(t, operator.NewPrincipal(
		operator.KindGUIAccount,
		"custom-read",
		[]string{operator.ResIPRules + ":" + operator.ActionRead},
	))
	if strings.Contains(body, `id="btnCreateRule"`) {
		t.Fatal("read-only page includes Create Rule")
	}
	if strings.Contains(body, `id="ipCheckForm"`) {
		t.Fatal("read-only page includes Check Access")
	}
	if !strings.Contains(body, `id="appFilter"`) {
		t.Fatal("read-only page missing app filter")
	}
}

func TestIPRulesPage_WriterKeepsWriteControls(t *testing.T) {
	body := renderIPRulesPage(t, operator.NewPrincipal(
		operator.KindGUIAccount,
		operator.RoleAdmin,
		operator.GrantsFor(operator.RoleAdmin),
	))
	if !strings.Contains(body, `id="btnCreateRule"`) {
		t.Fatal("writer missing Create Rule")
	}
	if !strings.Contains(body, `id="ipCheckForm"`) {
		t.Fatal("writer missing Check Access")
	}
}

func renderIPRulesPage(t *testing.T, principal operator.Principal) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.GET("/gui/ip-rules", func(c *gin.Context) {
		c.Request = httptest.NewRequest(http.MethodGet, "/gui/ip-rules", nil)
		data := web.AttachCan(web.TemplateData{ActivePage: "ip-rules"}, "/gui", principal.Has)
		c.HTML(http.StatusOK, "ip_rules", data)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/gui/ip-rules", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	return response.Body.String()
}
