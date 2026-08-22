package coreapp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
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

func TestGUIShell_ViewerOperatorIAMForbidden(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/operator", cookie, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
	users := guiGET(engine, "/gui/users", cookie, "", "")
	if strings.Contains(users.Body.String(), "Operator IAM") {
		t.Fatal("viewer nav includes Operator IAM")
	}
}

func TestGUIShell_AdminOperatorIAMForbidden(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDAdmin, operator.RoleAdmin)
	response := guiGET(engine, "/gui/operator", cookie, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
	users := guiGET(engine, "/gui/users", cookie, "", "")
	if strings.Contains(users.Body.String(), "Operator IAM") {
		t.Fatal("admin nav includes Operator IAM")
	}
}

func TestGUIShell_SuperadminOperatorIAMIncludesEnvKey(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(engine, "/gui/operator", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "env_key") {
		t.Fatalf("missing env_key: %s", body)
	}
	if !strings.Contains(body, "Operator IAM") {
		t.Fatal("superadmin nav missing Operator IAM")
	}
}

func TestGUIShell_ViewerOperatorIAMEventsForbidden(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(engine, "/gui/operator/iam-events", cookie, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
}

func TestGUIShell_SuperadminOperatorIAMEventsNewestFirst(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(engine, "/gui/operator/iam-events", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	assignAt := strings.Index(body, operator.ActionAssign)
	createAt := strings.Index(body, operator.ActionCreatePrincipal)
	if assignAt < 0 || createAt < 0 {
		t.Fatalf("missing events: %s", body)
	}
	if assignAt > createAt {
		t.Fatal("events are not newest first")
	}
	if !strings.Contains(body, "Operator IAM") {
		t.Fatal("typed events URL missing page chrome")
	}
}

func TestGUIShell_SuperadminAccessLogsDenyFilter(t *testing.T) {
	engine, cookie := guiShellEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(engine, "/gui/operator/access-logs?decision=deny", cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "/gui/tenants") {
		t.Fatalf("missing deny path: %s", body)
	}
	if strings.Contains(body, "<td>/gui/users</td>") {
		t.Fatal("deny filter included an allow row")
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

const lastSuperadminGUIMessage = "cannot demote or disable the last enabled superadmin"

func TestGUIShell_SuperadminOperatorIAMShowsWriteCTAs(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(fx.engine, "/gui/operator", fx.cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "Create operator") {
		t.Fatal("superadmin missing Create operator")
	}
	disablePath := "/gui/operator/accounts/" + fx.account.ID.String() + "/disable"
	if !strings.Contains(body, disablePath) {
		t.Fatalf("superadmin missing Disable for account: %s", body)
	}
	if strings.Contains(body, "/gui/operator/accounts//disable") {
		t.Fatal("env_key row has a Disable action")
	}
}

func TestGUIShell_SuperadminOperatorIAMIncludesRolesTab(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(fx.engine, "/gui/operator", fx.cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `data-tab="roles"`) {
		t.Fatal("superadmin missing Roles tab")
	}
}

func TestGUIShell_ViewerOperatorRolesForbidden(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(fx.engine, "/gui/operator/roles", fx.cookie, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGUIShell_SuperadminCreateCustomRoleAndAssign(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	formPage := guiGET(fx.engine, "/gui/operator/roles/new", fx.cookie, "", "")
	if formPage.Code != http.StatusOK {
		t.Fatalf("form status = %d, body = %s", formPage.Code, formPage.Body.String())
	}
	formBody := formPage.Body.String()
	if strings.Contains(formBody, operator.ResAdminIAM) {
		t.Fatal("custom role form includes admin_iam")
	}
	if !strings.Contains(formBody, `value="logs:read"`) {
		t.Fatalf("missing logs:read checkbox: %s", formBody)
	}

	created := guiPOST(fx.engine, "/gui/operator/roles", fx.cookie, url.Values{
		"name":   {"auditor"},
		"grants": {"logs:read"},
	})
	if created.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	var auditorID uuid.UUID
	for id, role := range fx.roles {
		if role.Name == "auditor" && !role.IsSystem {
			auditorID = id
		}
	}
	if auditorID == uuid.Nil {
		t.Fatal("auditor role was not stored")
	}
	p := operator.NewPrincipal(operator.KindAPIKey, "auditor", fx.roleGrants[auditorID])
	if !p.Has(operator.ResLogs, operator.ActionRead) {
		t.Fatal("auditor missing logs:read")
	}
	if p.Has(operator.ResTenants, operator.ActionWrite) {
		t.Fatal("auditor has tenants:write")
	}
	if p.Has(operator.ResAdminIAM, operator.ActionWrite) {
		t.Fatal("auditor has admin_iam:write")
	}

	assign := guiPUT(fx.engine, "/gui/operator/accounts/"+fx.viewerAccount.ID.String()+"/role", fx.cookie, url.Values{
		"operator_role_id": {auditorID.String()},
	})
	if assign.Code != http.StatusOK {
		t.Fatalf("assign status = %d, body = %s", assign.Code, assign.Body.String())
	}
	if fx.accounts[fx.viewerAccount.ID].OperatorRoleID != auditorID {
		t.Fatalf("assigned role = %s, want auditor", fx.accounts[fx.viewerAccount.ID].OperatorRoleID)
	}

	blocked := guiDELETE(fx.engine, "/gui/operator/roles/"+auditorID.String(), fx.cookie)
	if blocked.Code != http.StatusConflict {
		t.Fatalf("assigned delete status = %d, body = %s", blocked.Code, blocked.Body.String())
	}

	reassign := guiPUT(fx.engine, "/gui/operator/accounts/"+fx.viewerAccount.ID.String()+"/role", fx.cookie, url.Values{
		"operator_role_id": {operator.RoleIDViewer.String()},
	})
	if reassign.Code != http.StatusOK {
		t.Fatalf("reassign status = %d, body = %s", reassign.Code, reassign.Body.String())
	}
	deleted := guiDELETE(fx.engine, "/gui/operator/roles/"+auditorID.String(), fx.cookie)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleted.Code, deleted.Body.String())
	}
	if _, ok := fx.roles[auditorID]; ok {
		t.Fatal("auditor role still present after delete")
	}
}

func TestGUIShell_SuperadminCannotEditAdminRole(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(fx.engine, "/gui/operator/roles/"+operator.RoleIDAdmin.String()+"/edit", fx.cookie, "", "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), operator.ErrSystemRoleImmutable.Error()) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestGUIShell_TamperedAdminIAMGrantRejected(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiPOST(fx.engine, "/gui/operator/roles", fx.cookie, url.Values{
		"name":   {"rogue"},
		"grants": {"logs:read", "admin_iam:write"},
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), operator.ErrAdminIAMOnCustomRole.Error()) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestGUIShell_SuperadminOperatorAccountForm(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiGET(fx.engine, "/gui/operator/accounts/new", fx.cookie, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `name="username"`) || !strings.Contains(body, `name="password"`) {
		t.Fatalf("missing form fields: %s", body)
	}
}

func TestGUIShell_ViewerOperatorAccountFormForbidden(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDViewer, operator.RoleViewer)
	response := guiGET(fx.engine, "/gui/operator/accounts/new", fx.cookie, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
}

func TestGUIShell_SuperadminCreatesViewerOperator(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	form := url.Values{
		"username":         {"newviewer"},
		"email":            {"new@example.com"},
		"password":         {"twelvechars!!"},
		"operator_role_id": {operator.RoleIDSuperadmin.String()},
	}
	response := guiPOST(fx.engine, "/gui/operator/accounts", fx.cookie, form)
	if response.Code != http.StatusOK && response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(fx.created) != 1 {
		t.Fatalf("created = %d, want 1", len(fx.created))
	}
	if fx.created[0].OperatorRoleID != operator.RoleIDViewer {
		t.Fatalf("role = %s, want viewer", fx.created[0].OperatorRoleID)
	}
	if fx.created[0].PasswordHash == "" || fx.created[0].PasswordHash == "twelvechars!!" {
		t.Fatal("password was not hashed")
	}
	if len(fx.events) != 1 || fx.events[0].Action != operator.ActionCreatePrincipal {
		t.Fatalf("events = %+v", fx.events)
	}
	if fx.events[0].ActorKind != string(operator.KindGUIAccount) {
		t.Fatalf("actor_kind = %q", fx.events[0].ActorKind)
	}
	if fx.events[0].ActorAccountID == nil || *fx.events[0].ActorAccountID != fx.account.ID {
		t.Fatalf("actor_account_id = %v", fx.events[0].ActorAccountID)
	}
}

func TestGUIShell_LastSuperadminDisableIsHTML409(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	response := guiPOST(fx.engine, "/gui/operator/accounts/"+fx.account.ID.String()+"/disable", fx.cookie, url.Values{})
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	body := response.Body.String()
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("JSON body = %s", body)
	}
	if !strings.Contains(body, lastSuperadminGUIMessage) {
		t.Fatalf("body = %s", body)
	}
	if len(fx.disabled) != 0 {
		t.Fatal("last superadmin was disabled")
	}
}

func TestGUIShell_ViewerOperatorCreateForbidden(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDViewer, operator.RoleViewer)
	form := url.Values{
		"username": {"newviewer"},
		"password": {"twelvechars!!"},
	}
	response := guiPOST(fx.engine, "/gui/operator/accounts", fx.cookie, form)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != web.GUIForbiddenValue {
		t.Fatalf("missing %s", web.GUIForbiddenHeader)
	}
	if len(fx.created) != 0 {
		t.Fatal("viewer created an operator")
	}
	if len(fx.events) != 0 {
		t.Fatalf("viewer wrote IAM event: %+v", fx.events)
	}
}

func TestGUIShell_OperatorCreateMissingCSRFIsHTMLWithoutGUIHeader(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	form := url.Values{
		"username": {"newviewer"},
		"password": {"twelvechars!!"},
	}
	response := guiPOSTNoCSRF(fx.engine, "/gui/operator/accounts", fx.cookie, form)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get(web.GUIForbiddenHeader) != "" {
		t.Fatalf("CSRF 403 sent %s", web.GUIForbiddenHeader)
	}
	body := response.Body.String()
	if strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("JSON body = %s", body)
	}
	if !strings.Contains(body, "Reload this page and submit the form again.") {
		t.Fatalf("body = %s", body)
	}
	if len(fx.created) != 0 {
		t.Fatal("CSRF-missing POST created an operator")
	}
}

func TestGUIShell_SuperadminAssignsAccountRole(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	form := url.Values{"operator_role_id": {operator.RoleIDSupport.String()}}
	response := guiPUT(fx.engine, "/gui/operator/accounts/"+fx.viewerAccount.ID.String()+"/role", fx.cookie, form)
	if response.Code != http.StatusOK && response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if fx.accounts[fx.viewerAccount.ID].OperatorRoleID != operator.RoleIDSupport {
		t.Fatalf("role = %s", fx.accounts[fx.viewerAccount.ID].OperatorRoleID)
	}
	if len(fx.events) != 1 || fx.events[0].Action != operator.ActionAssign {
		t.Fatalf("events = %+v", fx.events)
	}
}

func TestGUIShell_SuperadminAssignsKeyRole(t *testing.T) {
	fx := newGUIShell(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin)
	form := url.Values{"operator_role_id": {operator.RoleIDSupport.String()}}
	response := guiPUT(fx.engine, "/gui/operator/keys/"+fx.viewerKey.ID.String()+"/role", fx.cookie, form)
	if response.Code != http.StatusOK && response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if fx.viewerKey.OperatorRoleID == nil || *fx.viewerKey.OperatorRoleID != operator.RoleIDSupport {
		t.Fatalf("key role = %v", fx.viewerKey.OperatorRoleID)
	}
	if len(fx.events) != 1 || fx.events[0].Action != operator.ActionAssign {
		t.Fatalf("events = %+v", fx.events)
	}
}

type guiShell struct {
	engine        *gin.Engine
	cookie        *http.Cookie
	account       *models.AdminAccount
	viewerAccount *models.AdminAccount
	viewerKey     *models.ApiKey
	accounts      map[uuid.UUID]*models.AdminAccount
	created       []*models.AdminAccount
	disabled      []uuid.UUID
	events        []operator.IAMEvent
	roles         map[uuid.UUID]sqlcgen.OperatorRole
	roleGrants    map[uuid.UUID][]string
}

func guiShellEngine(t *testing.T, roleID uuid.UUID, roleName string) (*gin.Engine, *http.Cookie) {
	t.Helper()
	fx := newGUIShell(t, roleID, roleName)
	return fx.engine, fx.cookie
}

func newGUIShell(t *testing.T, roleID uuid.UUID, roleName string) *guiShell {
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
	viewerAccount := &models.AdminAccount{
		ID:             uuid.New(),
		Username:       "ada",
		OperatorRoleID: operator.RoleIDViewer,
	}
	viewerRole := operator.RoleIDViewer
	viewerKey := &models.ApiKey{
		ID:             uuid.New(),
		KeyType:        admin.KeyTypeAdmin,
		Name:           "ops",
		OperatorRoleID: &viewerRole,
	}
	accounts := map[uuid.UUID]*models.AdminAccount{
		account.ID:       account,
		viewerAccount.ID: viewerAccount,
	}
	fx := &guiShell{
		account:       account,
		viewerAccount: viewerAccount,
		viewerKey:     viewerKey,
		accounts:      accounts,
		cookie:        &http.Cookie{Name: web.AdminSessionCookie, Value: "session-id"},
		roles: map[uuid.UUID]sqlcgen.OperatorRole{
			operator.RoleIDViewer:     {ID: operator.RoleIDViewer, Name: operator.RoleViewer, IsSystem: true},
			operator.RoleIDSupport:    {ID: operator.RoleIDSupport, Name: operator.RoleSupport, IsSystem: true},
			operator.RoleIDAdmin:      {ID: operator.RoleIDAdmin, Name: operator.RoleAdmin, IsSystem: true},
			operator.RoleIDSuperadmin: {ID: operator.RoleIDSuperadmin, Name: operator.RoleSuperadmin, IsSystem: true},
		},
		roleGrants: map[uuid.UUID][]string{
			operator.RoleIDViewer:     operator.GrantsFor(operator.RoleViewer),
			operator.RoleIDSupport:    operator.GrantsFor(operator.RoleSupport),
			operator.RoleIDAdmin:      operator.GrantsFor(operator.RoleAdmin),
			operator.RoleIDSuperadmin: operator.GrantsFor(operator.RoleSuperadmin),
		},
	}
	sessions := &stubGUISessions{account: account}
	grants := &stubGUIGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		roleID:                 {name: roleName, keys: operator.GrantsFor(roleName)},
		operator.RoleIDViewer:  {name: operator.RoleViewer, keys: operator.GrantsFor(operator.RoleViewer)},
		operator.RoleIDSupport: {name: operator.RoleSupport, keys: operator.GrantsFor(operator.RoleSupport)},
	}}
	h := &admin.GUIHandler{
		BasePath:       "/gui",
		AbortForbidden: middleware.AbortGUIForbidden,
		AbortInternal:  middleware.AbortGUIInternal,
		RoleExists: func(id uuid.UUID) (bool, error) {
			_, ok := fx.roles[id]
			return ok, nil
		},
		ListOperatorRoles: func(_ context.Context) ([]sqlcgen.OperatorRole, error) {
			out := make([]sqlcgen.OperatorRole, 0, len(fx.roles))
			for _, role := range fx.roles {
				out = append(out, role)
			}
			return out, nil
		},
		GetOperatorRole: func(_ context.Context, id uuid.UUID) (sqlcgen.OperatorRole, error) {
			role, ok := fx.roles[id]
			if !ok {
				return sqlcgen.OperatorRole{}, pgx.ErrNoRows
			}
			return role, nil
		},
		CreateOperatorRole: func(_ context.Context, name, description string, grantsList []operator.Permission) (sqlcgen.OperatorRole, error) {
			role := sqlcgen.OperatorRole{ID: uuid.New(), Name: name, Description: description, IsSystem: false}
			fx.roles[role.ID] = role
			keys := make([]string, 0, len(grantsList))
			for _, g := range grantsList {
				keys = append(keys, g.Key())
			}
			fx.roleGrants[role.ID] = keys
			grants.byID[role.ID] = struct {
				name string
				keys []string
			}{name: name, keys: keys}
			return role, nil
		},
		UpdateOperatorRole: func(_ context.Context, id uuid.UUID, name, description string) (sqlcgen.OperatorRole, error) {
			role, ok := fx.roles[id]
			if !ok {
				return sqlcgen.OperatorRole{}, pgx.ErrNoRows
			}
			if role.IsSystem {
				return sqlcgen.OperatorRole{}, operator.ErrSystemRoleImmutable
			}
			role.Name = name
			role.Description = description
			fx.roles[id] = role
			return role, nil
		},
		ReplaceOperatorRolePermissions: func(_ context.Context, id uuid.UUID, grants []operator.Permission) error {
			if _, ok := fx.roles[id]; !ok {
				return pgx.ErrNoRows
			}
			keys := make([]string, 0, len(grants))
			for _, g := range grants {
				keys = append(keys, g.Key())
			}
			fx.roleGrants[id] = keys
			return nil
		},
		DeleteOperatorRole: func(_ context.Context, id uuid.UUID) (int64, error) {
			role, ok := fx.roles[id]
			if !ok {
				return 0, pgx.ErrNoRows
			}
			if role.IsSystem {
				return 0, operator.ErrSystemRoleImmutable
			}
			for _, account := range fx.accounts {
				if account.OperatorRoleID == id {
					return 0, operator.ErrRoleAssigned
				}
			}
			if fx.viewerKey.OperatorRoleID != nil && *fx.viewerKey.OperatorRoleID == id {
				return 0, operator.ErrRoleAssigned
			}
			delete(fx.roles, id)
			delete(fx.roleGrants, id)
			return 1, nil
		},
		RoleGrantKeys: func(_ context.Context, id uuid.UUID) ([]string, error) {
			return fx.roleGrants[id], nil
		},
		RosterKeys: func() ([]operator.RosterEntry, error) {
			id := viewerKey.ID
			return []operator.RosterEntry{{
				Kind:        string(operator.KindAPIKey),
				DisplayName: viewerKey.Name,
				RoleName:    operator.RoleViewer,
				KeyID:       &id,
			}}, nil
		},
		RosterAccounts: func() ([]operator.RosterEntry, error) {
			sessionID := account.ID
			viewerID := viewerAccount.ID
			return []operator.RosterEntry{
				{
					Kind:        string(operator.KindGUIAccount),
					DisplayName: account.Username,
					RoleName:    roleName,
					AccountID:   &sessionID,
				},
				{
					Kind:        string(operator.KindGUIAccount),
					DisplayName: viewerAccount.Username,
					RoleName:    operator.RoleViewer,
					AccountID:   &viewerID,
				},
			}, nil
		},
		IAMEventList: func(_ context.Context, _ int32, _, _ *uuid.UUID) ([]operator.IAMEvent, error) {
			return []operator.IAMEvent{
				{Action: operator.ActionAssign, ActorKind: string(operator.KindAPIKey)},
				{Action: operator.ActionCreatePrincipal, ActorKind: string(operator.KindAPIKey)},
			}, nil
		},
		AccessLogList: func(_ context.Context, _ int32, decision *string) ([]operator.AccessRecord, error) {
			deny := operator.AccessRecord{Decision: operator.DecisionDeny, Path: "/gui/tenants", Method: http.MethodPost}
			allow := operator.AccessRecord{Decision: operator.DecisionAllow, Path: "/gui/users", Method: http.MethodGet}
			if decision != nil && *decision == operator.DecisionDeny {
				return []operator.AccessRecord{deny}, nil
			}
			return []operator.AccessRecord{deny, allow}, nil
		},
		RecordIAM: func(ev operator.IAMEvent) {
			fx.events = append(fx.events, ev)
		},
		CreateAccount: func(created *models.AdminAccount) error {
			if created.ID == uuid.Nil {
				created.ID = uuid.New()
			}
			copied := *created
			accounts[copied.ID] = &copied
			fx.created = append(fx.created, &copied)
			*created = copied
			return nil
		},
		GetAccount: func(id string) (*models.AdminAccount, error) {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return nil, err
			}
			stored, ok := accounts[parsed]
			if !ok {
				return nil, nil
			}
			copied := *stored
			if stored.DisabledAt != nil {
				disabled := *stored.DisabledAt
				copied.DisabledAt = &disabled
			}
			return &copied, nil
		},
		GetAccountByUsername: func(username string) (*models.AdminAccount, error) {
			for _, stored := range accounts {
				if stored.Username == username {
					copied := *stored
					return &copied, nil
				}
			}
			return nil, nil
		},
		UpdateAccountRole: func(id uuid.UUID, next uuid.UUID) error {
			stored, ok := accounts[id]
			if !ok {
				return pgx.ErrNoRows
			}
			stored.OperatorRoleID = next
			return nil
		},
		DisableAccount: func(id uuid.UUID) error {
			stored, ok := accounts[id]
			if !ok {
				return pgx.ErrNoRows
			}
			now := time.Now().UTC()
			stored.DisabledAt = &now
			fx.disabled = append(fx.disabled, id)
			return nil
		},
		CountEnabledSuperadmins: func() (int64, error) {
			var n int64
			for _, stored := range accounts {
				if stored.DisabledAt == nil && stored.OperatorRoleID == operator.RoleIDSuperadmin {
					n++
				}
			}
			return n, nil
		},
		GetAPIKey: func(id string) (*models.ApiKey, error) {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return nil, err
			}
			if parsed != viewerKey.ID {
				return nil, pgx.ErrNoRows
			}
			copied := *viewerKey
			if viewerKey.OperatorRoleID != nil {
				role := *viewerKey.OperatorRoleID
				copied.OperatorRoleID = &role
			}
			return &copied, nil
		},
		UpdateAPIKeyRole: func(id string, next *uuid.UUID) error {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return err
			}
			if parsed != viewerKey.ID {
				return pgx.ErrNoRows
			}
			if next == nil {
				viewerKey.OperatorRoleID = nil
				return nil
			}
			role := *next
			viewerKey.OperatorRoleID = &role
			return nil
		},
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
	guiAuth.GET("/operator", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorIAMPage)
	guiAuth.GET("/operator/roster", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorRosterList)
	guiAuth.GET("/operator/roster/export", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorRosterExport)
	guiAuth.GET("/operator/iam-events", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorIAMEvents)
	guiAuth.GET("/operator/iam-events/export", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorIAMEventsExport)
	guiAuth.GET("/operator/access-logs", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorAccessLogs)
	guiAuth.GET("/operator/access-logs/export", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorAccessLogsExport)
	guiAuth.GET("/operator/accounts/new", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorCreateAccountForm)
	guiAuth.POST("/operator/accounts", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorCreateAccount)
	guiAuth.PUT("/operator/accounts/:id/role", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorAccountRole)
	guiAuth.POST("/operator/accounts/:id/disable", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorDisableAccount)
	guiAuth.PUT("/operator/keys/:id/role", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorKeyRole)
	guiAuth.GET("/operator/roles", requireGUI(operator.ResAdminIAM, operator.ActionRead), h.OperatorRolesList)
	guiAuth.GET("/operator/roles/new", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorCreateRoleForm)
	guiAuth.POST("/operator/roles", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorCreateRole)
	guiAuth.GET("/operator/roles/:id/edit", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorEditRoleForm)
	guiAuth.PUT("/operator/roles/:id", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorUpdateRole)
	guiAuth.DELETE("/operator/roles/:id", requireGUI(operator.ResAdminIAM, operator.ActionWrite), h.OperatorDeleteRole)
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
	fx.engine = router
	return fx
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
	return guiForm(engine, http.MethodPost, path, cookie, form, "csrf-token")
}

func guiPOSTNoCSRF(engine *gin.Engine, path string, cookie *http.Cookie, form url.Values) *httptest.ResponseRecorder {
	return guiForm(engine, http.MethodPost, path, cookie, form, "")
}

func guiPUT(engine *gin.Engine, path string, cookie *http.Cookie, form url.Values) *httptest.ResponseRecorder {
	return guiForm(engine, http.MethodPut, path, cookie, form, "csrf-token")
}

func guiDELETE(engine *gin.Engine, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	return guiForm(engine, http.MethodDelete, path, cookie, url.Values{}, "csrf-token")
}

func guiForm(engine *gin.Engine, method, path string, cookie *http.Cookie, form url.Values, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	request.AddCookie(cookie)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}
