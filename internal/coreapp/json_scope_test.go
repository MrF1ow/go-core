package coreapp

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/geoip"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/internal/rbac"
	"github.com/MrF1ow/go-core/internal/twofa"
	"github.com/MrF1ow/go-core/internal/webhook"
	"github.com/MrF1ow/go-core/pkg/models"
)

const accessBoundKey = "access-bound-admin-key"

type jsonScopeHarness struct {
	engine        *gin.Engine
	boundAppID    uuid.UUID
	otherAppID    uuid.UUID
	homeUserID    uuid.UUID
	foreignUserID uuid.UUID
	gotExport     string
	createdKeys   int
}

func jsonScopeEngine(t *testing.T) *jsonScopeHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	boundApp := uuid.New()
	otherApp := uuid.New()
	homeUser := uuid.New()
	foreignUser := uuid.New()
	fx := &jsonScopeHarness{
		boundAppID:    boundApp,
		otherAppID:    otherApp,
		homeUserID:    homeUser,
		foreignUserID: foreignUser,
	}

	store := &accessLogKeyStore{}
	adminRole := operator.RoleIDAdmin
	superRole := operator.RoleIDSuperadmin
	boundKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &adminRole,
		AppID:          &boundApp,
	}
	superKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &superRole,
	}
	store.put(accessBoundKey, boundKey)
	store.put(accessSuperadminKey, superKey)
	grants := &accessLogGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		adminRole: {name: operator.RoleAdmin, keys: operator.GrantsFor(operator.RoleAdmin)},
		superRole: {name: operator.RoleSuperadmin, keys: operator.GrantsFor(operator.RoleSuperadmin)},
	}}

	handler := &admin.Handler{
		IPRuleRepo:        &geoip.IPRuleRepository{},
		TrustedDeviceRepo: &twofa.TrustedDeviceRepository{},
		GetUserDetail: func(id string) (*admin.UserDetail, error) {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return nil, err
			}
			switch parsed {
			case homeUser:
				return &admin.UserDetail{ID: parsed, AppID: boundApp}, nil
			case foreignUser:
				return &admin.UserDetail{ID: parsed, AppID: otherApp}, nil
			default:
				return nil, errJSONNotFound
			}
		},
		ExportUsersFn: func(appID, _ string) ([]admin.UserExportItem, bool, error) {
			fx.gotExport = appID
			return nil, false, nil
		},
		CreateAPIKey: func(key *models.ApiKey) error {
			fx.createdKeys++
			if key.ID == uuid.Nil {
				key.ID = uuid.New()
			}
			return nil
		},
	}

	engine := gin.New()
	group := engine.Group("/admin")
	group.Use(middleware.AdminAuthMiddleware("", store, grants))
	group.GET("/tenants", requireOp(operator.ResTenants, operator.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	group.POST("/operator/keys", requireOp(operator.ResAPIKeys, operator.ActionWrite), handler.OperatorCreateKey)
	group.GET("/apps/:id/ip-rules", requireOp(operator.ResIPRules, operator.ActionRead), handler.ListIPRules)
	group.POST("/apps/:id/oauth-config", requireOp(operator.ResOAuth, operator.ActionWrite), handler.UpsertOAuthConfig)
	group.GET("/users/export", requireOp(operator.ResUsers, operator.ActionRead), handler.ExportUsers)
	group.POST("/users/import", requireOp(operator.ResUsers, operator.ActionWrite), handler.ImportUsers)
	group.GET("/users/:id/trusted-devices", requireOp(operator.ResUsers, operator.ActionRead), handler.AdminListTrustedDevices)
	group.GET("/webhooks/apps/:app_id", requireOp(operator.ResWebhooks, operator.ActionRead), (&webhook.Handler{}).AdminListEndpointsByApp)
	group.POST("/rbac/roles", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), (&rbac.Handler{}).CreateRole)
	fx.engine = engine
	return fx
}

type jsonNotFound string

func (e jsonNotFound) Error() string { return string(e) }

const errJSONNotFound jsonNotFound = "not found"

func TestJSONScope_BoundTenantsForbidden(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodGet, "/admin/tenants", accessBoundKey, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestJSONScope_PlatformTenantsOK(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodGet, "/admin/tenants", accessSuperadminKey, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestJSONScope_BoundMintForbidden(t *testing.T) {
	fx := jsonScopeEngine(t)
	body := `{"name":"ci","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	rec := iamEventDo(fx.engine, http.MethodPost, "/admin/operator/keys", accessBoundKey, body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if fx.createdKeys != 0 {
		t.Fatal("bound mint inserted a key")
	}
}

func TestJSONScope_BoundIPRulesForeignNotFound(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodGet, "/admin/apps/"+fx.otherAppID.String()+"/ip-rules", accessBoundKey, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"error":"not found"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestJSONScope_BoundOAuthForeignNotFound(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodPost, "/admin/apps/"+fx.otherAppID.String()+"/oauth-config", accessBoundKey, `{"provider":"google","client_id":"x","client_secret":"y"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestJSONScope_BoundExportUsesBoundApp(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodGet, "/admin/users/export?format=json&app_id="+fx.otherAppID.String(), accessBoundKey, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if fx.gotExport != fx.boundAppID.String() {
		t.Fatalf("export app = %q, want bound %s", fx.gotExport, fx.boundAppID)
	}
	if strings.Contains(rec.Body.String(), fx.otherAppID.String()) {
		t.Fatalf("export leaked other app: %s", rec.Body.String())
	}
}

func TestJSONScope_PlatformExportKeepsRequestedApp(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodGet, "/admin/users/export?format=json&app_id="+fx.otherAppID.String(), accessSuperadminKey, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if fx.gotExport != fx.otherAppID.String() {
		t.Fatalf("export app = %q, want requested %s", fx.gotExport, fx.otherAppID)
	}
}

func TestJSONScope_BoundImportForeignNotFound(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodPost, "/admin/users/import?app_id="+fx.otherAppID.String(), accessBoundKey, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestJSONScope_BoundTrustedDeviceForeignNotFound(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodGet, "/admin/users/"+fx.foreignUserID.String()+"/trusted-devices", accessBoundKey, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestJSONScope_BoundWebhooksForeignNotFound(t *testing.T) {
	fx := jsonScopeEngine(t)
	rec := iamEventDo(fx.engine, http.MethodGet, "/admin/webhooks/apps/"+fx.otherAppID.String(), accessBoundKey, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestJSONScope_BoundRBACCreateForeignNotFound(t *testing.T) {
	fx := jsonScopeEngine(t)
	body := `{"app_id":"` + fx.otherAppID.String() + `","name":"x"}`
	rec := iamEventDo(fx.engine, http.MethodPost, "/admin/rbac/roles", accessBoundKey, body)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
