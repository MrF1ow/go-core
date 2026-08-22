package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/web"
)

func TestTenantListView_AdminCanWrite(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	p := adminPrincipal()
	c.Set(web.OperatorPrincipalKey, &p)
	view := (&GUIHandler{}).tenantListView(c, nil, 1, 1, 0)
	if !view.CanWrite {
		t.Fatal("admin list missing CanWrite")
	}
}

func TestTenantListView_ViewerCannotWrite(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleViewer, operator.GrantsFor(operator.RoleViewer))
	c.Set(web.OperatorPrincipalKey, &p)
	view := (&GUIHandler{}).tenantListView(c, nil, 1, 1, 0)
	if view.CanWrite {
		t.Fatal("viewer list has CanWrite")
	}
}

func TestTenantListTemplate_WriteCTAsFollowCanWrite(t *testing.T) {
	item := TenantListItem{
		ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Name:      "acme",
		AppCount:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	withWrite := renderTenantList(t, tenantListData{
		Tenants:    []TenantListItem{item},
		Page:       1,
		TotalPages: 1,
		Total:      1,
		CanWrite:   true,
	})
	if !strings.Contains(withWrite, "bi-pencil") || !strings.Contains(withWrite, "bi-trash") {
		t.Fatalf("writer missing row actions: %s", withWrite)
	}
	withoutWrite := renderTenantList(t, tenantListData{
		Tenants:    []TenantListItem{item},
		Page:       1,
		TotalPages: 1,
		Total:      1,
		CanWrite:   false,
	})
	if strings.Contains(withoutWrite, "bi-pencil") || strings.Contains(withoutWrite, "bi-trash") {
		t.Fatalf("reader saw row actions: %s", withoutWrite)
	}
}

func renderTenantList(t *testing.T, data tenantListData) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.GET("/list", func(c *gin.Context) {
		c.HTML(http.StatusOK, "tenant_list", data)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/list", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "acme") {
		t.Fatalf("truncated or empty body = %s", body)
	}
	return body
}
