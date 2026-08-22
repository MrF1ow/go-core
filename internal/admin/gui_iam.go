package admin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/internal/safeconv"
	"github.com/MrF1ow/go-core/pkg/models"
)

const (
	operatorTabRoster     = "roster"
	operatorTabEvents     = "events"
	operatorTabAccessLogs = "access-logs"
)

func (h *GUIHandler) writeIAMEvent(c *gin.Context, ev operator.IAMEvent) {
	if ev.ActorKind == "" {
		if p, ok := guiPrincipal(c); ok {
			ev.ActorKind = string(p.Kind)
			ev.ActorKeyID = p.KeyID
			ev.ActorAccountID = p.AccountID
		}
	}
	if h.RecordIAM != nil {
		h.RecordIAM(ev)
		return
	}
	if h.OperatorRepo != nil {
		if err := h.OperatorRepo.InsertIAMEvent(context.Background(), ev); err != nil {
			log.Printf("operator IAM event insert: %v", err)
		}
	}
}

func uuidPtrsEqual(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (h *GUIHandler) loadRoster() ([]operator.RosterEntry, bool, error) {
	return loadOperatorRoster(h.rosterKeys, h.rosterAccounts)
}

func (h *GUIHandler) rosterKeys() ([]operator.RosterEntry, error) {
	return rosterKeysFrom(h.RosterKeys, h.Repo)
}

func (h *GUIHandler) rosterAccounts() ([]operator.RosterEntry, error) {
	var list func() ([]models.AdminAccount, error)
	if h.AccountService != nil && h.AccountService.Repo != nil {
		list = h.AccountService.Repo.ListAll
	}
	return rosterAccountsFrom(h.RosterAccounts, list, h.accountRoleName)
}

func (h *GUIHandler) accountRoleName(account models.AdminAccount) string {
	return operatorAccountRoleName(h.OperatorRepo, account)
}

type operatorIAMView struct {
	Tab       string
	Entries   []rosterViewEntry
	Truncated bool
	Events    []operator.IAMEvent
	Logs      []operator.AccessRecord
	CanWrite  bool
	Roles     []operatorRoleOption
}

type rosterViewEntry struct {
	operator.RosterEntry
	Status string
}

func rosterStatus(entry operator.RosterEntry) string {
	if entry.Revoked {
		return "Revoked"
	}
	if entry.Disabled != nil && *entry.Disabled {
		return "Disabled"
	}
	return "Active"
}

func (h *GUIHandler) rosterPageData() (operatorIAMView, error) {
	entries, truncated, err := h.loadRoster()
	if err != nil {
		return operatorIAMView{}, err
	}
	rows := make([]rosterViewEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, rosterViewEntry{RosterEntry: entry, Status: rosterStatus(entry)})
	}
	return operatorIAMView{Tab: operatorTabRoster, Entries: rows, Truncated: truncated}, nil
}

func (h *GUIHandler) rosterView(c *gin.Context) (operatorIAMView, error) {
	view, err := h.rosterPageData()
	if err != nil {
		return operatorIAMView{}, err
	}
	view.CanWrite = h.principalCan(c, operator.ResAdminIAM, operator.ActionWrite)
	view.Roles = operatorRoleOptions()
	return view, nil
}

func (h *GUIHandler) operatorIAMHTML(c *gin.Context, view operatorIAMView) {
	data := h.page(c)
	data.ActivePage = "operator-iam"
	data.Data = view
	c.HTML(http.StatusOK, "operator_iam", data)
}

func (h *GUIHandler) operatorPanel(c *gin.Context, fragment string, view operatorIAMView) {
	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, fragment, view)
		return
	}
	h.operatorIAMHTML(c, view)
}

func (h *GUIHandler) listIAMEvents(ctx context.Context, limit int32, targetKeyID, targetAccountID *uuid.UUID) ([]operator.IAMEvent, error) {
	return listOperatorIAMEvents(ctx, h.IAMEventList, h.OperatorRepo, limit, targetKeyID, targetAccountID)
}

func (h *GUIHandler) listAccessLogs(ctx context.Context, limit int32, decision *string) ([]operator.AccessRecord, error) {
	return listOperatorAccessLogs(ctx, h.AccessLogList, h.OperatorRepo, limit, decision)
}

func parseOptionalUUIDQuery(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseDecisionQuery(raw string) (*string, error) {
	if raw == "" {
		return nil, nil
	}
	if raw != operator.DecisionAllow && raw != operator.DecisionDeny {
		return nil, fmt.Errorf("decision must be allow or deny")
	}
	return &raw, nil
}

// OperatorIAMPage renders the read-only operator roster.
// GET /gui/operator
func (h *GUIHandler) OperatorIAMPage(c *gin.Context) {
	roster, err := h.rosterView(c)
	if err != nil {
		log.Printf("operator IAM page: %v", err)
		h.abortInternal(c)
		return
	}
	h.operatorIAMHTML(c, roster)
}

// OperatorRosterList returns the roster table partial (HTMX fragment).
// GET /gui/operator/roster
func (h *GUIHandler) OperatorRosterList(c *gin.Context) {
	roster, err := h.rosterView(c)
	if err != nil {
		log.Printf("operator IAM roster list: %v", err)
		h.abortInternal(c)
		return
	}
	c.HTML(http.StatusOK, "operator_roster", roster)
}

// OperatorRosterExport streams the roster as CSV.
// GET /gui/operator/roster/export
func (h *GUIHandler) OperatorRosterExport(c *gin.Context) {
	entries, truncated, err := h.loadRoster()
	if err != nil {
		log.Printf("operator IAM roster export: %v", err)
		h.abortInternal(c)
		return
	}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, rosterCSVRow(entry))
	}
	writeOperatorCSV(c, "operator-roster.csv", truncated, rosterCSVHeader, rows)
}

// OperatorIAMEvents renders IAM events newest first.
// GET /gui/operator/iam-events
func (h *GUIHandler) OperatorIAMEvents(c *gin.Context) {
	targetKeyID, err := parseOptionalUUIDQuery(c.Query("target_key_id"))
	if err != nil {
		h.abortInternal(c)
		return
	}
	targetAccountID, err := parseOptionalUUIDQuery(c.Query("target_account_id"))
	if err != nil {
		h.abortInternal(c)
		return
	}
	entries, err := h.listIAMEvents(c.Request.Context(), parseOperatorListLimit(c.Query("limit")), targetKeyID, targetAccountID)
	if err != nil {
		log.Printf("operator IAM events: %v", err)
		h.abortInternal(c)
		return
	}
	h.operatorPanel(c, "operator_iam_events", operatorIAMView{Tab: operatorTabEvents, Events: entries})
}

// OperatorIAMEventsExport streams IAM events as CSV.
// GET /gui/operator/iam-events/export
func (h *GUIHandler) OperatorIAMEventsExport(c *gin.Context) {
	entries, err := h.listIAMEvents(c.Request.Context(), safeconv.ToInt32(operator.ExportMaxRows+1), nil, nil)
	if err != nil {
		log.Printf("operator IAM events export: %v", err)
		h.abortInternal(c)
		return
	}
	entries, truncated := capExport(entries)
	rows := make([][]string, 0, len(entries))
	for _, ev := range entries {
		rows = append(rows, iamEventCSVRow(ev))
	}
	writeOperatorCSV(c, "operator-iam-events.csv", truncated, iamEventCSVHeader, rows)
}

// OperatorAccessLogs renders operator access logs.
// GET /gui/operator/access-logs
func (h *GUIHandler) OperatorAccessLogs(c *gin.Context) {
	decision, err := parseDecisionQuery(c.Query("decision"))
	if err != nil {
		h.abortInternal(c)
		return
	}
	entries, err := h.listAccessLogs(c.Request.Context(), parseOperatorListLimit(c.Query("limit")), decision)
	if err != nil {
		log.Printf("operator access logs: %v", err)
		h.abortInternal(c)
		return
	}
	h.operatorPanel(c, "operator_access_logs", operatorIAMView{Tab: operatorTabAccessLogs, Logs: entries})
}

// OperatorAccessLogsExport streams access logs as CSV.
// GET /gui/operator/access-logs/export
func (h *GUIHandler) OperatorAccessLogsExport(c *gin.Context) {
	entries, err := h.listAccessLogs(c.Request.Context(), safeconv.ToInt32(operator.ExportMaxRows+1), nil)
	if err != nil {
		log.Printf("operator access logs export: %v", err)
		h.abortInternal(c)
		return
	}
	entries, truncated := capExport(entries)
	rows := make([][]string, 0, len(entries))
	for _, rec := range entries {
		rows = append(rows, accessLogCSVRow(rec))
	}
	writeOperatorCSV(c, "operator-access-logs.csv", truncated, accessLogCSVHeader, rows)
}

func guiOperatorAlert(c *gin.Context, status int, kind, message string) {
	c.String(status, `<div class="alert alert-`+kind+` alert-dismissible fade show" role="alert">`+escapeHTML(message)+`<button type="button" class="btn-close" data-bs-dismiss="alert"></button></div>`)
}

func guiLastSuperadminConflict(c *gin.Context) {
	guiOperatorAlert(c, http.StatusConflict, "danger", lastSuperadminMessage)
}

// OperatorCreateAccountForm renders the create-operator form fragment.
// GET /gui/operator/accounts/new
func (h *GUIHandler) OperatorCreateAccountForm(c *gin.Context) {
	c.HTML(http.StatusOK, "operator_account_form", gin.H{})
}

// OperatorCreateAccount creates a GUI operator as viewer.
// POST /gui/operator/accounts
func (h *GUIHandler) OperatorCreateAccount(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	if len(username) < 3 || len(username) > 50 {
		guiOperatorAlert(c, http.StatusBadRequest, "danger", "Username must be between 3 and 50 characters.")
		return
	}
	if len(password) < 12 || len(password) > 128 {
		guiOperatorAlert(c, http.StatusBadRequest, "danger", "Password must be between 12 and 128 characters.")
		return
	}

	existing, err := h.guiLookupAccountByUsername(username)
	if err != nil {
		log.Printf("operator GUI create account lookup: %v", err)
		h.abortInternal(c)
		return
	}
	if existing != nil {
		guiOperatorAlert(c, http.StatusConflict, "danger", "username already exists")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("operator GUI create account hash: %v", err)
		h.abortInternal(c)
		return
	}
	roleID := operator.RoleIDViewer
	account := &models.AdminAccount{
		Username:       username,
		Email:          email,
		PasswordHash:   string(hashed),
		OperatorRoleID: roleID,
	}
	if err := h.guiCreateAccount(account); err != nil {
		log.Printf("operator GUI create account: %v", err)
		h.abortInternal(c)
		return
	}
	if account.ID == uuid.Nil {
		log.Printf("operator GUI create account: missing id")
		h.abortInternal(c)
		return
	}
	accountID := account.ID
	newRole := roleID
	h.writeIAMEvent(c, operator.IAMEvent{
		TargetKind:      operator.KindGUIAccount,
		TargetAccountID: &accountID,
		NewRoleID:       &newRole,
		Action:          operator.ActionCreatePrincipal,
	})
	guiOperatorAlert(c, http.StatusOK, "success", "Operator created successfully.")
}

// OperatorAccountRole assigns a system operator role to a GUI account.
// PUT /gui/operator/accounts/:id/role
func (h *GUIHandler) OperatorAccountRole(c *gin.Context) {
	account, ok := h.guiLoadAccountOrAbort(c, c.Param("id"))
	if !ok {
		return
	}
	roleID, err := uuid.Parse(strings.TrimSpace(c.PostForm("operator_role_id")))
	if err != nil || !operator.IsSystemRoleID(roleID) {
		guiOperatorAlert(c, http.StatusBadRequest, "danger", "Invalid operator role.")
		return
	}
	if account.OperatorRoleID == roleID {
		c.Status(http.StatusNoContent)
		return
	}
	if roleID != operator.RoleIDSuperadmin {
		blocked, err := h.guiWouldLeaveLastSuperadmin(account)
		if err != nil {
			log.Printf("operator GUI account role: %v", err)
			h.abortInternal(c)
			return
		}
		if blocked {
			guiLastSuperadminConflict(c)
			return
		}
	}
	if err := h.guiUpdateAccountRole(account.ID, roleID); err != nil {
		log.Printf("operator GUI account role: %v", err)
		h.abortInternal(c)
		return
	}
	oldRole := account.OperatorRoleID
	newRole := roleID
	accountID := account.ID
	h.writeIAMEvent(c, operator.IAMEvent{
		TargetKind:      operator.KindGUIAccount,
		TargetAccountID: &accountID,
		OldRoleID:       &oldRole,
		NewRoleID:       &newRole,
		Action:          operator.ActionAssign,
	})
	guiOperatorAlert(c, http.StatusOK, "success", "Operator role updated.")
}

// OperatorDisableAccount sets disabled_at on a GUI account.
// POST /gui/operator/accounts/:id/disable
func (h *GUIHandler) OperatorDisableAccount(c *gin.Context) {
	account, ok := h.guiLoadAccountOrAbort(c, c.Param("id"))
	if !ok {
		return
	}
	if account.DisabledAt != nil {
		c.Status(http.StatusNoContent)
		return
	}
	blocked, err := h.guiWouldLeaveLastSuperadmin(account)
	if err != nil {
		log.Printf("operator GUI disable account: %v", err)
		h.abortInternal(c)
		return
	}
	if blocked {
		guiLastSuperadminConflict(c)
		return
	}
	if err := h.guiDisableAccount(account.ID); err != nil {
		log.Printf("operator GUI disable account: %v", err)
		h.abortInternal(c)
		return
	}
	accountID := account.ID
	h.writeIAMEvent(c, operator.IAMEvent{
		TargetKind:      operator.KindGUIAccount,
		TargetAccountID: &accountID,
		Action:          operator.ActionDisablePrincipal,
	})
	guiOperatorAlert(c, http.StatusOK, "success", "Operator account disabled.")
}

// OperatorKeyRole assigns an operator role to an admin API key.
// PUT /gui/operator/keys/:id/role
func (h *GUIHandler) OperatorKeyRole(c *gin.Context) {
	id := c.Param("id")
	key, err := h.guiLoadAPIKey(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			guiOperatorAlert(c, http.StatusNotFound, "danger", "API key not found.")
			return
		}
		if _, parseErr := uuid.Parse(id); parseErr != nil {
			guiOperatorAlert(c, http.StatusBadRequest, "danger", "Invalid API key ID.")
			return
		}
		log.Printf("operator GUI key role load: %v", err)
		h.abortInternal(c)
		return
	}
	if key == nil {
		guiOperatorAlert(c, http.StatusNotFound, "danger", "API key not found.")
		return
	}
	if key.KeyType != KeyTypeAdmin {
		guiOperatorAlert(c, http.StatusBadRequest, "danger", "operator role applies to admin keys only")
		return
	}

	p, ok := guiPrincipal(c)
	if !ok {
		h.abortInternal(c)
		return
	}
	roleID, err := operator.ParseAssignedAdminRole(*p, c.PostForm("operator_role_id"), key.KeyType, key.OperatorRoleID)
	if errors.Is(err, operator.ErrIAMAssignmentDenied) {
		h.abortForbidden(c)
		return
	}
	if err != nil {
		guiOperatorAlert(c, http.StatusBadRequest, "danger", "Invalid operator role.")
		return
	}
	if uuidPtrsEqual(key.OperatorRoleID, roleID) {
		c.Status(http.StatusNoContent)
		return
	}
	var oldRole *uuid.UUID
	if key.OperatorRoleID != nil {
		copied := *key.OperatorRoleID
		oldRole = &copied
	}
	if err := h.guiUpdateAPIKeyRole(id, key, roleID); err != nil {
		log.Printf("operator GUI key role: %v", err)
		h.abortInternal(c)
		return
	}
	keyID := key.ID
	h.writeIAMEvent(c, operator.IAMEvent{
		TargetKind:  operator.KindAPIKey,
		TargetKeyID: &keyID,
		OldRoleID:   oldRole,
		NewRoleID:   roleID,
		Action:      operator.ActionAssign,
	})
	guiOperatorAlert(c, http.StatusOK, "success", "API key role updated.")
}

func (h *GUIHandler) guiLookupAccountByUsername(username string) (*models.AdminAccount, error) {
	if h.GetAccountByUsername != nil {
		return h.GetAccountByUsername(username)
	}
	if h.AccountService != nil && h.AccountService.Repo != nil {
		return h.AccountService.Repo.GetByUsername(username)
	}
	return nil, nil
}

func (h *GUIHandler) guiCreateAccount(account *models.AdminAccount) error {
	if h.CreateAccount != nil {
		return h.CreateAccount(account)
	}
	if h.AccountService == nil || h.AccountService.Repo == nil {
		return fmt.Errorf("account repository is not configured")
	}
	return h.AccountService.Repo.Create(account)
}

func (h *GUIHandler) guiLoadAccount(id string) (*models.AdminAccount, error) {
	if h.GetAccount != nil {
		return h.GetAccount(id)
	}
	if h.AccountService == nil || h.AccountService.Repo == nil {
		return nil, fmt.Errorf("account repository is not configured")
	}
	return h.AccountService.Repo.GetByID(id)
}

func (h *GUIHandler) guiLoadAccountOrAbort(c *gin.Context, id string) (*models.AdminAccount, bool) {
	account, err := h.guiLoadAccount(id)
	if err != nil {
		if _, parseErr := uuid.Parse(id); parseErr != nil {
			guiOperatorAlert(c, http.StatusBadRequest, "danger", "Invalid account ID.")
			return nil, false
		}
		if errors.Is(err, pgx.ErrNoRows) {
			guiOperatorAlert(c, http.StatusNotFound, "danger", "Operator account not found.")
			return nil, false
		}
		log.Printf("operator GUI load account: %v", err)
		h.abortInternal(c)
		return nil, false
	}
	if account == nil {
		guiOperatorAlert(c, http.StatusNotFound, "danger", "Operator account not found.")
		return nil, false
	}
	return account, true
}

func (h *GUIHandler) guiUpdateAccountRole(id uuid.UUID, roleID uuid.UUID) error {
	if h.UpdateAccountRole != nil {
		return h.UpdateAccountRole(id, roleID)
	}
	if h.AccountService == nil || h.AccountService.Repo == nil {
		return fmt.Errorf("account repository is not configured")
	}
	return h.AccountService.Repo.UpdateOperatorRole(id, roleID)
}

func (h *GUIHandler) guiDisableAccount(id uuid.UUID) error {
	if h.DisableAccount != nil {
		return h.DisableAccount(id)
	}
	if h.AccountService == nil || h.AccountService.Repo == nil {
		return fmt.Errorf("account repository is not configured")
	}
	now := time.Now().UTC()
	return h.AccountService.Repo.SetDisabledAt(id, &now)
}

func (h *GUIHandler) guiCountEnabledSuperadmins() (int64, error) {
	if h.CountEnabledSuperadmins != nil {
		return h.CountEnabledSuperadmins()
	}
	if h.AccountService == nil || h.AccountService.Repo == nil {
		return 0, fmt.Errorf("account repository is not configured")
	}
	return h.AccountService.Repo.CountEnabledSuperadmins(context.Background())
}

func (h *GUIHandler) guiWouldLeaveLastSuperadmin(account *models.AdminAccount) (bool, error) {
	targetIsEnabledSuperadmin := account.DisabledAt == nil && account.OperatorRoleID == operator.RoleIDSuperadmin
	count, err := h.guiCountEnabledSuperadmins()
	if err != nil {
		return false, err
	}
	return operator.WouldLeaveLastSuperadmin(int(count), targetIsEnabledSuperadmin), nil
}

func (h *GUIHandler) guiLoadAPIKey(id string) (*models.ApiKey, error) {
	if h.GetAPIKey != nil {
		return h.GetAPIKey(id)
	}
	return h.loadAPIKey(id)
}

func (h *GUIHandler) guiUpdateAPIKeyRole(id string, key *models.ApiKey, roleID *uuid.UUID) error {
	if h.UpdateAPIKeyRole != nil {
		return h.UpdateAPIKeyRole(id, roleID)
	}
	return h.persistAPIKeyUpdate(id, key.Name, key.Description, key.Scopes, roleID, key.ExpiresAt)
}
