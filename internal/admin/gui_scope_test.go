package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

func scopeContext(t *testing.T, appID *uuid.UUID) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/gui/users", nil)
	p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleAdmin, operator.GrantsFor(operator.RoleAdmin))
	p.AppID = appID
	c.Set(web.OperatorPrincipalKey, &p)
	return c
}

func TestRestrictAppQuery_UnboundKeepsRequested(t *testing.T) {
	c := scopeContext(t, nil)
	got := restrictAppQuery(c, "requested-app")
	if got != "requested-app" {
		t.Fatalf("restrictAppQuery = %q, want requested-app", got)
	}
}

func TestRestrictAppQuery_BoundOverwritesRequested(t *testing.T) {
	bound := uuid.New()
	c := scopeContext(t, &bound)
	got := restrictAppQuery(c, uuid.New().String())
	if got != bound.String() {
		t.Fatalf("restrictAppQuery = %q, want %s", got, bound)
	}
	if restrictAppQuery(c, "") != bound.String() {
		t.Fatal("bound empty request should still return bound app")
	}
}

func TestForeignApp(t *testing.T) {
	bound := uuid.New()
	other := uuid.New()
	c := scopeContext(t, &bound)
	if !foreignApp(c, other) {
		t.Fatal("foreignApp should be true for another app")
	}
	if foreignApp(c, bound) {
		t.Fatal("foreignApp should be false for the bound app")
	}
	unbound := scopeContext(t, nil)
	if foreignApp(unbound, other) {
		t.Fatal("foreignApp should be false when unbound")
	}
}

func TestApiKeyForeign_BoundRejectsAdmin(t *testing.T) {
	bound := uuid.New()
	c := scopeContext(t, &bound)
	adminKey := &models.ApiKey{KeyType: KeyTypeAdmin}
	if !apiKeyForeign(c, adminKey) {
		t.Fatal("bound operator must not see admin keys")
	}
	appKey := &models.ApiKey{KeyType: KeyTypeApp, AppID: &bound}
	if apiKeyForeign(c, appKey) {
		t.Fatal("bound operator should see app keys for their app")
	}
	other := uuid.New()
	otherKey := &models.ApiKey{KeyType: KeyTypeApp, AppID: &other}
	if !apiKeyForeign(c, otherKey) {
		t.Fatal("bound operator must not see another app's keys")
	}
	if apiKeyForeign(scopeContext(t, nil), adminKey) {
		t.Fatal("unbound should see admin keys")
	}
}

func TestUserList_BoundOverwritesRequestedApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	bound := uuid.New()
	other := uuid.New()
	var gotApp string
	h := &GUIHandler{
		BasePath: "/gui",
		ListUsers: func(_, _ int, appID, _ string) ([]UserListItem, int64, error) {
			gotApp = appID
			return nil, 0, nil
		},
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.GET("/gui/users/list", func(c *gin.Context) {
		p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleAdmin, operator.GrantsFor(operator.RoleAdmin))
		p.AppID = &bound
		c.Set(web.OperatorPrincipalKey, &p)
		h.UserList(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/gui/users/list?app_id="+other.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if gotApp != bound.String() {
		t.Fatalf("UserList appID = %q, want bound %s", gotApp, bound)
	}
}

func TestUserList_UnboundKeepsRequestedApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	requested := uuid.New()
	var gotApp string
	h := &GUIHandler{
		BasePath: "/gui",
		ListUsers: func(_, _ int, appID, _ string) ([]UserListItem, int64, error) {
			gotApp = appID
			return nil, 0, nil
		},
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.GET("/gui/users/list", func(c *gin.Context) {
		p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleAdmin, operator.GrantsFor(operator.RoleAdmin))
		c.Set(web.OperatorPrincipalKey, &p)
		h.UserList(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/gui/users/list?app_id="+requested.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if gotApp != requested.String() {
		t.Fatalf("UserList appID = %q, want requested %s", gotApp, requested)
	}
}

func TestUserDetail_ForeignAppNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	bound := uuid.New()
	other := uuid.New()
	homeID := uuid.New()
	foreignID := uuid.New()
	h := &GUIHandler{
		BasePath: "/gui",
		GetUserDetail: func(id string) (*UserDetail, error) {
			parsed, parseErr := uuid.Parse(id)
			if parseErr != nil {
				return nil, parseErr
			}
			switch parsed {
			case homeID:
				return &UserDetail{ID: parsed, AppID: bound}, nil
			case foreignID:
				return &UserDetail{ID: parsed, AppID: other}, nil
			default:
				return nil, errNotFound
			}
		},
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.GET("/gui/users/:id", func(c *gin.Context) {
		p := operator.NewPrincipal(operator.KindGUIAccount, operator.RoleAdmin, operator.GrantsFor(operator.RoleAdmin))
		p.AppID = &bound
		c.Set(web.OperatorPrincipalKey, &p)
		h.UserDetail(c)
	})

	foreign := httptest.NewRecorder()
	router.ServeHTTP(foreign, httptest.NewRequest(http.MethodGet, "/gui/users/"+foreignID.String(), nil))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign status = %d, body = %s", foreign.Code, foreign.Body.String())
	}

	home := httptest.NewRecorder()
	router.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/gui/users/"+homeID.String(), nil))
	if home.Code != http.StatusOK {
		t.Fatalf("home status = %d, body = %s", home.Code, home.Body.String())
	}
}
