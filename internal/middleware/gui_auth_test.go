package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

type stubGUISessions struct {
	account *models.AdminAccount
	err     error
}

func (s *stubGUISessions) ValidateSession(string) (*models.AdminAccount, error) {
	return s.account, s.err
}

func (s *stubGUISessions) GenerateCSRFToken(string) (string, error) {
	return "", nil
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

func TestGUIAuth_ViewerAccountAttachesPrincipal(t *testing.T) {
	accountID := uuid.New()
	account := &models.AdminAccount{
		ID:             accountID,
		Username:       "viewer",
		OperatorRoleID: operator.RoleIDViewer,
	}
	grants := guiGrants(operator.RoleIDViewer, operator.RoleViewer)

	response, principal := guiGET(t, &stubGUISessions{account: account}, grants)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if principal == nil {
		t.Fatal("missing operator principal")
	}
	if principal.Kind != operator.KindGUIAccount {
		t.Fatalf("kind = %q, want %q", principal.Kind, operator.KindGUIAccount)
	}
	if principal.RoleName != operator.RoleViewer {
		t.Fatalf("role = %q, want %q", principal.RoleName, operator.RoleViewer)
	}
	if principal.Has(operator.ResTenants, operator.ActionWrite) {
		t.Fatal("viewer has tenants:write")
	}
	if !principal.Has(operator.ResDashboard, operator.ActionRead) {
		t.Fatal("viewer lacks dashboard:read")
	}
	if principal.AccountID == nil || *principal.AccountID != accountID {
		t.Fatalf("account ID = %v, want %s", principal.AccountID, accountID)
	}
	if principal.AppID != nil {
		t.Fatalf("app ID = %s, want nil", *principal.AppID)
	}
	if principal.KeyID != nil {
		t.Fatalf("key ID = %s, want nil", *principal.KeyID)
	}
}

func TestGUIAuth_BoundViewerAttachesAppID(t *testing.T) {
	accountID := uuid.New()
	appID := uuid.New()
	account := &models.AdminAccount{
		ID:             accountID,
		Username:       "bound-viewer",
		OperatorRoleID: operator.RoleIDViewer,
		AppID:          &appID,
	}
	grants := guiGrants(operator.RoleIDViewer, operator.RoleViewer)

	response, principal := guiGET(t, &stubGUISessions{account: account}, grants)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if principal == nil {
		t.Fatal("missing operator principal")
	}
	if principal.AppID == nil || *principal.AppID != appID {
		t.Fatalf("app ID = %v, want %s", principal.AppID, appID)
	}
	if principal.AccountID == nil || *principal.AccountID != accountID {
		t.Fatalf("account ID = %v, want %s", principal.AccountID, accountID)
	}
}

func TestGUIAuth_SuperadminAccountHasAdminIAMWrite(t *testing.T) {
	account := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "root",
		OperatorRoleID: operator.RoleIDSuperadmin,
	}
	grants := guiGrants(operator.RoleIDSuperadmin, operator.RoleSuperadmin)

	response, principal := guiGET(t, &stubGUISessions{account: account}, grants)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if principal == nil || !principal.Has(operator.ResAdminIAM, operator.ActionWrite) {
		t.Fatal("superadmin principal lacks admin_iam:write")
	}
}

func TestGUIAuth_NilAccountRedirectsToLogin(t *testing.T) {
	response, principal := guiGET(t, &stubGUISessions{}, guiGrants(operator.RoleIDViewer, operator.RoleViewer))

	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}
	if location := response.Header().Get("Location"); location != "/gui/login?redirect=/gui/private" {
		t.Fatalf("location = %q", location)
	}
	if principal != nil {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if cookie := response.Header().Get("Set-Cookie"); !strings.Contains(cookie, web.AdminSessionCookie+"=") || !strings.Contains(cookie, "Max-Age=0") {
		t.Fatalf("cleared cookie header = %q", cookie)
	}
}

func TestGUIAuth_UnknownRoleReturns500WithoutPrincipal(t *testing.T) {
	account := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "unknown",
		OperatorRoleID: uuid.New(),
	}

	response, principal := guiGET(t, &stubGUISessions{account: account}, &stubGUIGrants{})

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if principal != nil {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content type = %q, want HTML or plain text", response.Header().Get("Content-Type"))
	}
}

func TestGUIAuth_ZeroRoleReturns500WithoutPrincipal(t *testing.T) {
	account := &models.AdminAccount{ID: uuid.New(), Username: "missing-role"}

	response, principal := guiGET(t, &stubGUISessions{account: account}, guiGrants(operator.RoleIDViewer, operator.RoleViewer))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if principal != nil {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestGUIAuth_MissingGrantLookupReturns500(t *testing.T) {
	account := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "viewer",
		OperatorRoleID: operator.RoleIDViewer,
	}

	response, principal := guiGET(t, &stubGUISessions{account: account}, nil)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if principal != nil {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestGUIAuth_InvalidSessionClearsCookieAndRedirects(t *testing.T) {
	response, principal := guiGET(t, &stubGUISessions{err: errors.New("expired")}, nil)

	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}
	if principal != nil {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if cookie := response.Header().Get("Set-Cookie"); !strings.Contains(cookie, web.AdminSessionCookie+"=") || !strings.Contains(cookie, "Max-Age=0") {
		t.Fatalf("cleared cookie header = %q", cookie)
	}
}

func TestGUIAuth_DisabledAccountRedirectsToLogin(t *testing.T) {
	account := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "disabled",
		OperatorRoleID: operator.RoleIDSuperadmin,
		DisabledAt:     timePtr(time.Now()),
	}
	grants := guiGrants(operator.RoleIDSuperadmin, operator.RoleSuperadmin)

	response, principal := guiGET(t, &stubGUISessions{account: account}, grants)

	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}
	if location := response.Header().Get("Location"); location != "/gui/login?redirect=/gui/private" {
		t.Fatalf("location = %q", location)
	}
	if principal != nil {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if cookie := response.Header().Get("Set-Cookie"); !strings.Contains(cookie, web.AdminSessionCookie+"=") || !strings.Contains(cookie, "Max-Age=0") {
		t.Fatalf("cleared cookie header = %q", cookie)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func guiGrants(roleID uuid.UUID, roleName string) *stubGUIGrants {
	return &stubGUIGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		roleID: {name: roleName, keys: operator.GrantsFor(roleName)},
	}}
}

func guiGET(t *testing.T, sessions web.SessionValidator, grants operator.GrantLookup) (*httptest.ResponseRecorder, *operator.Principal) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var principal *operator.Principal
	router.Use(func(c *gin.Context) {
		c.Next()
		value, ok := c.Get(web.OperatorPrincipalKey)
		if ok {
			principal, _ = value.(*operator.Principal)
		}
	})
	router.GET(
		"/gui/private",
		GUIAuthMiddleware(sessions, grants, "/gui"),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/gui/private", nil)
	request.AddCookie(&http.Cookie{Name: web.AdminSessionCookie, Value: "session-id"})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	return response, principal
}
