package admin

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/internal/safeconv"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/pkg/dto"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

type rosterResponse struct {
	Entries []operator.RosterEntry `json:"entries"`
}

// OperatorRoster lists admin keys and GUI accounts plus the synthetic env-key row.
// @Summary Operator IAM roster
// @Tags Admin
// @Produce json
// @Success 200 {object} rosterResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/roster [get]
func (h *Handler) OperatorRoster(c *gin.Context) {
	entries, _, err := h.loadRoster()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to load operator roster"})
		return
	}
	c.JSON(http.StatusOK, rosterResponse{Entries: entries})
}

// OperatorRosterExport exports the roster as CSV.
// @Summary Export operator IAM roster
// @Tags Admin
// @Produce text/csv
// @Success 200 {string} string "CSV export"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/roster/export [get]
func (h *Handler) OperatorRosterExport(c *gin.Context) {
	entries, truncated, err := h.loadRoster()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to load operator roster"})
		return
	}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, rosterCSVRow(entry))
	}
	writeOperatorCSV(c, "operator-roster.csv", truncated, rosterCSVHeader, rows)
}

var rosterCSVHeader = []string{"kind", "id", "display_name", "role", "created_at", "last_used_at", "expires_at", "revoked", "disabled"}

func (h *Handler) loadRoster() ([]operator.RosterEntry, bool, error) {
	return loadOperatorRoster(h.rosterKeys, h.rosterAccounts)
}

func loadOperatorRoster(keysFn, accountsFn func() ([]operator.RosterEntry, error)) ([]operator.RosterEntry, bool, error) {
	keys, err := keysFn()
	if err != nil {
		return nil, false, err
	}
	accounts, err := accountsFn()
	if err != nil {
		return nil, false, err
	}
	raw := 1 + len(keys) + len(accounts)
	return operator.BuildRoster(operator.EnvKeyRosterEntry(), keys, accounts), raw > operator.ExportMaxRows, nil
}

func (h *Handler) rosterKeys() ([]operator.RosterEntry, error) {
	return rosterKeysFrom(h.RosterKeys, h.Repo)
}

func rosterKeysFrom(override func() ([]operator.RosterEntry, error), repo *Repository) ([]operator.RosterEntry, error) {
	if override != nil {
		return override()
	}
	if repo == nil {
		return nil, fmt.Errorf("api key repository is not configured")
	}
	items, _, err := repo.ListApiKeys(1, operator.ExportMaxRows, KeyTypeAdmin)
	if err != nil {
		return nil, err
	}
	out := make([]operator.RosterEntry, 0, len(items))
	for i := range items {
		item := items[i]
		id := item.ID
		out = append(out, operator.RosterEntry{
			Kind:        string(operator.KindAPIKey),
			DisplayName: item.Name,
			RoleName:    item.OperatorRoleName,
			KeyID:       &id,
			CreatedAt:   item.CreatedAt,
			LastUsedAt:  item.LastUsedAt,
			ExpiresAt:   item.ExpiresAt,
			Revoked:     item.IsRevoked,
		})
	}
	return out, nil
}

func (h *Handler) rosterAccounts() ([]operator.RosterEntry, error) {
	var list func() ([]models.AdminAccount, error)
	if h.Accounts != nil {
		list = h.Accounts.ListAll
	}
	return rosterAccountsFrom(h.RosterAccounts, list, h.accountRoleName)
}

func rosterAccountsFrom(override func() ([]operator.RosterEntry, error), list func() ([]models.AdminAccount, error), roleName func(models.AdminAccount) string) ([]operator.RosterEntry, error) {
	if override != nil {
		return override()
	}
	if list == nil {
		return nil, fmt.Errorf("account repository is not configured")
	}
	accounts, err := list()
	if err != nil {
		return nil, err
	}
	out := make([]operator.RosterEntry, 0, len(accounts))
	for i := range accounts {
		out = append(out, accountRosterEntry(accounts[i], roleName(accounts[i])))
	}
	return out, nil
}

func accountRosterEntry(account models.AdminAccount, roleName string) operator.RosterEntry {
	id := account.ID
	disabled := account.DisabledAt != nil
	return operator.RosterEntry{
		Kind:        string(operator.KindGUIAccount),
		DisplayName: account.Username,
		RoleName:    roleName,
		AccountID:   &id,
		CreatedAt:   account.CreatedAt,
		LastUsedAt:  account.LastLoginAt,
		Disabled:    &disabled,
	}
}

func (h *Handler) accountRoleName(account models.AdminAccount) string {
	return operatorAccountRoleName(h.OperatorRoles, account)
}

func operatorAccountRoleName(roles *operator.Repository, account models.AdminAccount) string {
	if name, ok := operator.RoleNameForID(account.OperatorRoleID); ok {
		return name
	}
	if roles == nil {
		return ""
	}
	name, err := roles.RoleName(context.Background(), account.OperatorRoleID)
	if err != nil {
		log.Printf("operator roster role lookup for account %s: %v", account.ID, err)
		return ""
	}
	return name
}

func rosterCSVRow(entry operator.RosterEntry) []string {
	id := ""
	switch {
	case entry.KeyID != nil:
		id = entry.KeyID.String()
	case entry.AccountID != nil:
		id = entry.AccountID.String()
	}
	disabled := ""
	if entry.Disabled != nil {
		disabled = fmt.Sprintf("%t", *entry.Disabled)
	}
	return []string{
		entry.Kind,
		id,
		entry.DisplayName,
		entry.RoleName,
		csvTime(entry.CreatedAt),
		csvTimePtr(entry.LastUsedAt),
		csvTimePtr(entry.ExpiresAt),
		fmt.Sprintf("%t", entry.Revoked),
		disabled,
	}
}

func csvTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func csvTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return csvTime(*value)
}

func csvUUID(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

func csvUUIDPtr(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func capExport[T any](items []T) ([]T, bool) {
	if len(items) > operator.ExportMaxRows {
		return items[:operator.ExportMaxRows], true
	}
	return items, false
}

func writeOperatorCSV(c *gin.Context, filename string, truncated bool, header []string, rows [][]string) {
	if truncated {
		c.Header("X-Export-Truncated", "true")
	} else {
		c.Header("X-Export-Truncated", "false")
	}
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Status(http.StatusOK)
	writer := csv.NewWriter(c.Writer)
	if err := writer.Write(header); err != nil {
		log.Printf("operator csv header %s: %v", filename, err)
		return
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			log.Printf("operator csv row %s: %v", filename, err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("operator csv flush %s: %v", filename, err)
	}
}

var accessLogCSVHeader = []string{
	"id", "at", "kind", "key_id", "account_id", "role_name", "method", "path", "decision", "resource", "action", "status",
	"ip_address", "user_agent",
}

func accessLogCSVRow(rec operator.AccessRecord) []string {
	return []string{
		csvUUID(rec.ID),
		csvTime(rec.At),
		string(rec.Kind),
		csvUUIDPtr(rec.KeyID),
		csvUUIDPtr(rec.AccountID),
		rec.RoleName,
		rec.Method,
		rec.Path,
		rec.Decision,
		rec.Resource,
		rec.Action,
		strconv.Itoa(rec.Status),
		rec.IPAddress,
		rec.UserAgent,
	}
}

var iamEventCSVHeader = []string{
	"id", "at", "actor_kind", "actor_key_id", "actor_account_id",
	"target_kind", "target_key_id", "target_account_id", "old_role_id", "new_role_id", "action",
}

func iamEventCSVRow(ev operator.IAMEvent) []string {
	return []string{
		csvUUID(ev.ID),
		csvTime(ev.At),
		ev.ActorKind,
		csvUUIDPtr(ev.ActorKeyID),
		csvUUIDPtr(ev.ActorAccountID),
		string(ev.TargetKind),
		csvUUIDPtr(ev.TargetKeyID),
		csvUUIDPtr(ev.TargetAccountID),
		csvUUIDPtr(ev.OldRoleID),
		csvUUIDPtr(ev.NewRoleID),
		ev.Action,
	}
}

type accessLogResponse struct {
	Entries []operator.AccessRecord `json:"entries"`
}

func parseOperatorListLimit(raw string) int32 {
	limit := int32(100)
	if raw == "" {
		return limit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return limit
	}
	if n > 1000 {
		n = 1000
	}
	return safeconv.ToInt32(n)
}

// OperatorAccessLogs lists recorded operator permission decisions.
// @Summary List operator access logs
// @Description Newest-first JSON list of operator allow and deny decisions
// @Tags Admin
// @Produce json
// @Param limit query int false "Max rows, default 100, max 1000" default(100)
// @Param decision query string false "Filter by allow or deny"
// @Success 200 {object} accessLogResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/access-logs [get]
func (h *Handler) OperatorAccessLogs(c *gin.Context) {
	limit := parseOperatorListLimit(c.Query("limit"))

	var decision *string
	if raw := c.Query("decision"); raw != "" {
		if raw != operator.DecisionAllow && raw != operator.DecisionDeny {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "decision must be allow or deny"})
			return
		}
		decision = &raw
	}

	entries, err := h.listAccessLogs(c.Request.Context(), limit, decision)
	if err != nil {
		log.Printf("operator access logs: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to list operator access logs"})
		return
	}
	if entries == nil {
		entries = []operator.AccessRecord{}
	}
	c.JSON(http.StatusOK, accessLogResponse{Entries: entries})
}

// OperatorAccessLogsExport exports access logs as CSV.
// @Summary Export operator access logs
// @Tags Admin
// @Produce text/csv
// @Success 200 {string} string "CSV export"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/access-logs/export [get]
func (h *Handler) OperatorAccessLogsExport(c *gin.Context) {
	entries, err := h.listAccessLogs(c.Request.Context(), safeconv.ToInt32(operator.ExportMaxRows+1), nil)
	if err != nil {
		log.Printf("operator access logs export: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to list operator access logs"})
		return
	}
	entries, truncated := capExport(entries)
	rows := make([][]string, 0, len(entries))
	for _, rec := range entries {
		rows = append(rows, accessLogCSVRow(rec))
	}
	writeOperatorCSV(c, "operator-access-logs.csv", truncated, accessLogCSVHeader, rows)
}

func (h *Handler) listAccessLogs(ctx context.Context, limit int32, decision *string) ([]operator.AccessRecord, error) {
	return listOperatorAccessLogs(ctx, h.AccessLogList, h.OperatorRoles, limit, decision)
}

func listOperatorAccessLogs(ctx context.Context, override func(context.Context, int32, *string) ([]operator.AccessRecord, error), repo *operator.Repository, limit int32, decision *string) ([]operator.AccessRecord, error) {
	if override != nil {
		return override(ctx, limit, decision)
	}
	if repo == nil {
		return nil, fmt.Errorf("operator repository is not configured")
	}
	return repo.ListAccessLogs(ctx, limit, 0, decision)
}

type iamEventResponse struct {
	Entries []operator.IAMEvent `json:"entries"`
}

// OperatorIAMEvents lists recorded operator IAM events.
// @Summary List operator IAM events
// @Description Newest-first JSON list of operator role assignment and principal lifecycle events
// @Tags Admin
// @Produce json
// @Param limit query int false "Max rows, default 100, max 1000" default(100)
// @Param target_key_id query string false "Filter by target API key UUID"
// @Param target_account_id query string false "Filter by target GUI account UUID"
// @Success 200 {object} iamEventResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/iam-events [get]
func (h *Handler) OperatorIAMEvents(c *gin.Context) {
	limit := parseOperatorListLimit(c.Query("limit"))

	var targetKeyID *uuid.UUID
	if raw := c.Query("target_key_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "target_key_id must be a UUID"})
			return
		}
		targetKeyID = &id
	}

	var targetAccountID *uuid.UUID
	if raw := c.Query("target_account_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "target_account_id must be a UUID"})
			return
		}
		targetAccountID = &id
	}

	entries, err := h.listIAMEvents(c.Request.Context(), limit, targetKeyID, targetAccountID)
	if err != nil {
		log.Printf("operator IAM events: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to list operator IAM events"})
		return
	}
	if entries == nil {
		entries = []operator.IAMEvent{}
	}
	c.JSON(http.StatusOK, iamEventResponse{Entries: entries})
}

// OperatorIAMEventsExport exports IAM events as CSV.
// @Summary Export operator IAM events
// @Tags Admin
// @Produce text/csv
// @Success 200 {string} string "CSV export"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/iam-events/export [get]
func (h *Handler) OperatorIAMEventsExport(c *gin.Context) {
	entries, err := h.listIAMEvents(c.Request.Context(), safeconv.ToInt32(operator.ExportMaxRows+1), nil, nil)
	if err != nil {
		log.Printf("operator IAM events export: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to list operator IAM events"})
		return
	}
	entries, truncated := capExport(entries)
	rows := make([][]string, 0, len(entries))
	for _, ev := range entries {
		rows = append(rows, iamEventCSVRow(ev))
	}
	writeOperatorCSV(c, "operator-iam-events.csv", truncated, iamEventCSVHeader, rows)
}

func (h *Handler) listIAMEvents(ctx context.Context, limit int32, targetKeyID, targetAccountID *uuid.UUID) ([]operator.IAMEvent, error) {
	return listOperatorIAMEvents(ctx, h.IAMEventList, h.OperatorRoles, limit, targetKeyID, targetAccountID)
}

func listOperatorIAMEvents(ctx context.Context, override func(context.Context, int32, *uuid.UUID, *uuid.UUID) ([]operator.IAMEvent, error), repo *operator.Repository, limit int32, targetKeyID, targetAccountID *uuid.UUID) ([]operator.IAMEvent, error) {
	if override != nil {
		return override(ctx, limit, targetKeyID, targetAccountID)
	}
	if repo == nil {
		return nil, fmt.Errorf("operator repository is not configured")
	}
	return repo.ListIAMEvents(ctx, limit, 0, targetKeyID, targetAccountID)
}

type assignKeyRoleRequest struct {
	OperatorRoleID string `json:"operator_role_id" binding:"required"`
}

// OperatorKeyRole assigns an operator role to an admin API key.
// @Summary Assign operator role on an admin API key
// @Description Stamp operator_role_id on an existing admin key. Same role is 204 with no event.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "API key UUID"
// @Param request body assignKeyRoleRequest true "Role assignment"
// @Success 204 "Role assigned or unchanged"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/keys/{id}/role [put]
func (h *Handler) OperatorKeyRole(c *gin.Context) {
	id := c.Param("id")
	var req assignKeyRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	key, err := h.loadOperatorAPIKey(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "API key not found"})
			return
		}
		if _, parseErr := uuid.Parse(id); parseErr != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid API key ID"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to load API key"})
		return
	}
	if key == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "API key not found"})
		return
	}
	if key.KeyType != KeyTypeAdmin {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "operator role applies to admin keys only"})
		return
	}

	p, ok := jsonPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal authentication error"})
		return
	}
	if strings.TrimSpace(req.OperatorRoleID) == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator role"})
		return
	}
	roleID, err := operator.ParseAssignedAdminRole(*p, req.OperatorRoleID, key.KeyType, key.OperatorRoleID, h.roleExists())
	if errors.Is(err, operator.ErrIAMAssignmentDenied) {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator role"})
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
	if err := h.updateOperatorAPIKeyRole(id, key, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to update API key role"})
		return
	}
	keyID := key.ID
	h.writeJSONIAMEvent(c, operator.IAMEvent{
		TargetKind:  operator.KindAPIKey,
		TargetKeyID: &keyID,
		OldRoleID:   oldRole,
		NewRoleID:   roleID,
		Action:      operator.ActionAssign,
	})
	c.Status(http.StatusNoContent)
}

func jsonPrincipal(c *gin.Context) (*operator.Principal, bool) {
	val, ok := c.Get(web.OperatorPrincipalKey)
	if !ok {
		return nil, false
	}
	p, ok := val.(*operator.Principal)
	return p, ok && p != nil
}

func (h *Handler) writeJSONIAMEvent(c *gin.Context, ev operator.IAMEvent) {
	if ev.ActorKind == "" {
		if p, ok := jsonPrincipal(c); ok {
			ev.ActorKind = string(p.Kind)
			ev.ActorKeyID = p.KeyID
			ev.ActorAccountID = p.AccountID
		}
	}
	var err error
	if h.IAMEventWrite != nil {
		err = h.IAMEventWrite(ev)
	} else if h.OperatorRoles != nil {
		err = h.OperatorRoles.InsertIAMEvent(context.Background(), ev)
	}
	if err != nil {
		log.Printf("operator IAM event insert: %v", err)
	}
}

func (h *Handler) loadOperatorAPIKey(id string) (*models.ApiKey, error) {
	if h.GetAPIKey != nil {
		return h.GetAPIKey(id)
	}
	if h.Repo != nil {
		return h.Repo.GetApiKeyByID(id)
	}
	return nil, fmt.Errorf("api key repository is not configured")
}

func (h *Handler) updateOperatorAPIKeyRole(id string, key *models.ApiKey, roleID *uuid.UUID) error {
	if h.UpdateAPIKeyRole != nil {
		return h.UpdateAPIKeyRole(id, roleID)
	}
	if h.Repo == nil {
		return fmt.Errorf("api key repository is not configured")
	}
	return h.Repo.UpdateApiKey(id, key.Name, key.Description, key.Scopes, roleID, key.ExpiresAt)
}

const lastSuperadminMessage = "cannot demote or disable the last enabled superadmin"

type createOperatorAccountRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required,min=12,max=128"`
}

type operatorAccountResponse struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	OperatorRoleID uuid.UUID `json:"operator_role_id"`
	Role           string    `json:"role"`
}

type assignAccountRoleRequest struct {
	OperatorRoleID string `json:"operator_role_id" binding:"required"`
}

// OperatorCreateAccount creates a GUI operator as viewer.
// @Summary Create an operator GUI account
// @Description Creates a GUI operator with role viewer. Posted roles are ignored.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body createOperatorAccountRequest true "Account"
// @Success 201 {object} operatorAccountResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/accounts [post]
func (h *Handler) OperatorCreateAccount(c *gin.Context) {
	var req createOperatorAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	existing, err := h.lookupAccountByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create operator account"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "username already exists"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create operator account"})
		return
	}
	roleID := operator.RoleIDViewer
	account := &models.AdminAccount{
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   string(hashed),
		OperatorRoleID: roleID,
	}
	if err := h.createOperatorAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create operator account"})
		return
	}
	if account.ID == uuid.Nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create operator account"})
		return
	}
	accountID := account.ID
	newRole := roleID
	h.writeJSONIAMEvent(c, operator.IAMEvent{
		TargetKind:      operator.KindGUIAccount,
		TargetAccountID: &accountID,
		NewRoleID:       &newRole,
		Action:          operator.ActionCreatePrincipal,
	})
	c.JSON(http.StatusCreated, operatorAccountResponse{
		ID:             account.ID,
		Username:       account.Username,
		Email:          account.Email,
		OperatorRoleID: operator.RoleIDViewer,
		Role:           operator.RoleViewer,
	})
}

// OperatorAccountRole assigns an operator role to a GUI account.
// @Summary Assign operator role on a GUI account
// @Description Refuses with 409 when the change would leave zero enabled superadmin accounts
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "Account UUID"
// @Param request body assignAccountRoleRequest true "Role assignment"
// @Success 204 "Role assigned or unchanged"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/accounts/{id}/role [put]
func (h *Handler) OperatorAccountRole(c *gin.Context) {
	id := c.Param("id")
	var req assignAccountRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	if strings.TrimSpace(req.OperatorRoleID) == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator role"})
		return
	}
	account, ok := h.loadOperatorAccountOrAbort(c, id)
	if !ok {
		return
	}
	p, ok := jsonPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal authentication error"})
		return
	}
	current := account.OperatorRoleID
	roleID, err := operator.ParseAssignedAdminRole(*p, req.OperatorRoleID, KeyTypeAdmin, &current, h.roleExists())
	if errors.Is(err, operator.ErrIAMAssignmentDenied) {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: "insufficient permissions"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator role"})
		return
	}
	if roleID == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator role"})
		return
	}
	if account.OperatorRoleID == *roleID {
		c.Status(http.StatusNoContent)
		return
	}
	if *roleID != operator.RoleIDSuperadmin {
		blocked, err := h.wouldLeaveLastSuperadmin(account)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to update operator account"})
			return
		}
		if blocked {
			c.JSON(http.StatusConflict, dto.ErrorResponse{Error: lastSuperadminMessage})
			return
		}
	}
	if err := h.updateOperatorAccountRole(account.ID, *roleID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to update operator account"})
		return
	}
	oldRole := account.OperatorRoleID
	accountID := account.ID
	h.writeJSONIAMEvent(c, operator.IAMEvent{
		TargetKind:      operator.KindGUIAccount,
		TargetAccountID: &accountID,
		OldRoleID:       &oldRole,
		NewRoleID:       roleID,
		Action:          operator.ActionAssign,
	})
	c.Status(http.StatusNoContent)
}

// OperatorDisableAccount sets disabled_at on a GUI account.
// @Summary Disable an operator GUI account
// @Description Idempotent if already disabled. Refuses the last enabled superadmin with 409.
// @Tags Admin
// @Produce json
// @Param id path string true "Account UUID"
// @Success 204 "Disabled or already disabled"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/accounts/{id}/disable [post]
func (h *Handler) OperatorDisableAccount(c *gin.Context) {
	id := c.Param("id")
	account, ok := h.loadOperatorAccountOrAbort(c, id)
	if !ok {
		return
	}
	if account.DisabledAt != nil {
		c.Status(http.StatusNoContent)
		return
	}
	blocked, err := h.wouldLeaveLastSuperadmin(account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to disable operator account"})
		return
	}
	if blocked {
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: lastSuperadminMessage})
		return
	}
	if err := h.disableOperatorAccount(account.ID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to disable operator account"})
		return
	}
	accountID := account.ID
	h.writeJSONIAMEvent(c, operator.IAMEvent{
		TargetKind:      operator.KindGUIAccount,
		TargetAccountID: &accountID,
		Action:          operator.ActionDisablePrincipal,
	})
	c.Status(http.StatusNoContent)
}

type operatorRoleListResponse struct {
	Roles []operatorRoleJSON `json:"roles"`
}

type operatorRoleJSON struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	Grants      []string  `json:"grants,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type createOperatorRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Grants      []string `json:"grants"`
}

type updateOperatorRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type replaceOperatorRolePermissionsRequest struct {
	Grants []string `json:"grants"`
}

// OperatorListRoles lists seeded and custom operator roles.
// @Summary List operator roles
// @Tags Admin
// @Produce json
// @Success 200 {object} operatorRoleListResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/roles [get]
func (h *Handler) OperatorListRoles(c *gin.Context) {
	if h.OperatorRoles == nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to list operator roles"})
		return
	}
	roles, err := h.OperatorRoles.ListRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to list operator roles"})
		return
	}
	out := make([]operatorRoleJSON, 0, len(roles))
	for _, role := range roles {
		out = append(out, operatorRoleJSON{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, operatorRoleListResponse{Roles: out})
}

// OperatorCreateRole creates a custom operator role.
// @Summary Create a custom operator role
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body createOperatorRoleRequest true "Role"
// @Success 201 {object} operatorRoleJSON
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/roles [post]
func (h *Handler) OperatorCreateRole(c *gin.Context) {
	var req createOperatorRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "name is required"})
		return
	}
	if operator.IsReservedSystemRoleName(name) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: operator.ErrReservedSystemRoleName.Error()})
		return
	}
	grants, ok := parsePostedCustomGrants(c, req.Grants)
	if !ok {
		return
	}
	var (
		role sqlcgen.OperatorRole
		err  error
	)
	if h.CreateOperatorRole != nil {
		role, err = h.CreateOperatorRole(c.Request.Context(), name, strings.TrimSpace(req.Description), grants)
	} else if h.OperatorRoles != nil {
		role, err = h.OperatorRoles.CreateCustomRole(c.Request.Context(), name, strings.TrimSpace(req.Description), grants)
	} else {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create operator role"})
		return
	}
	if err != nil {
		writeOperatorRoleWriteError(c, err, "Failed to create operator role")
		return
	}
	c.JSON(http.StatusCreated, operatorRoleJSON{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		Grants:      grantKeys(grants),
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
}

// OperatorUpdateRole updates name and description on a custom operator role.
// @Summary Update a custom operator role
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "Role UUID"
// @Param request body updateOperatorRoleRequest true "Role"
// @Success 200 {object} operatorRoleJSON
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/roles/{id} [put]
func (h *Handler) OperatorUpdateRole(c *gin.Context) {
	id, ok := h.loadMutableOperatorRole(c)
	if !ok {
		return
	}
	var req updateOperatorRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "name is required"})
		return
	}
	if operator.IsReservedSystemRoleName(name) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: operator.ErrReservedSystemRoleName.Error()})
		return
	}
	updated, err := h.OperatorRoles.UpdateCustomRole(c.Request.Context(), id, name, strings.TrimSpace(req.Description))
	if err != nil {
		writeOperatorRoleWriteError(c, err, "Failed to update operator role")
		return
	}
	c.JSON(http.StatusOK, operatorRoleJSON{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		IsSystem:    updated.IsSystem,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	})
}

// OperatorReplaceRolePermissions replaces grants on a custom operator role.
// @Summary Replace custom operator role grants
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "Role UUID"
// @Param request body replaceOperatorRolePermissionsRequest true "Grants"
// @Success 204 "Grants replaced"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/roles/{id}/permissions [put]
func (h *Handler) OperatorReplaceRolePermissions(c *gin.Context) {
	id, ok := h.loadMutableOperatorRole(c)
	if !ok {
		return
	}
	var req replaceOperatorRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	grants, ok := parsePostedCustomGrants(c, req.Grants)
	if !ok {
		return
	}
	if err := h.OperatorRoles.ReplaceRolePermissions(c.Request.Context(), id, grants); err != nil {
		writeOperatorRoleWriteError(c, err, "Failed to update operator role grants")
		return
	}
	c.Status(http.StatusNoContent)
}

// OperatorDeleteRole deletes a custom operator role.
// @Summary Delete a custom operator role
// @Tags Admin
// @Produce json
// @Param id path string true "Role UUID"
// @Success 204 "Deleted"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security AdminApiKey
// @Router /admin/operator/roles/{id} [delete]
func (h *Handler) OperatorDeleteRole(c *gin.Context) {
	id, ok := h.loadMutableOperatorRole(c)
	if !ok {
		return
	}
	n, err := h.OperatorRoles.DeleteCustomRole(c.Request.Context(), id)
	if err != nil {
		writeOperatorRoleWriteError(c, err, "Failed to delete operator role")
		return
	}
	if n == 0 {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Operator role not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) loadOperatorAccountOrAbort(c *gin.Context, id string) (*models.AdminAccount, bool) {
	account, err := h.loadOperatorAccount(id)
	if err != nil {
		if _, parseErr := uuid.Parse(id); parseErr != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid account ID"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to load operator account"})
		return nil, false
	}
	if account == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Operator account not found"})
		return nil, false
	}
	return account, true
}

func (h *Handler) wouldLeaveLastSuperadmin(account *models.AdminAccount) (bool, error) {
	targetIsEnabledSuperadmin := account.DisabledAt == nil && account.OperatorRoleID == operator.RoleIDSuperadmin
	count, err := h.countEnabledSuperadmins()
	if err != nil {
		return false, err
	}
	return operator.WouldLeaveLastSuperadmin(int(count), targetIsEnabledSuperadmin), nil
}

func (h *Handler) lookupAccountByUsername(username string) (*models.AdminAccount, error) {
	if h.Accounts != nil {
		return h.Accounts.GetByUsername(username)
	}
	return nil, nil
}

func (h *Handler) createOperatorAccount(account *models.AdminAccount) error {
	if h.CreateAccount != nil {
		return h.CreateAccount(account)
	}
	if h.Accounts == nil {
		return fmt.Errorf("account repository is not configured")
	}
	return h.Accounts.Create(account)
}

func (h *Handler) loadOperatorAccount(id string) (*models.AdminAccount, error) {
	if h.GetAccount != nil {
		return h.GetAccount(id)
	}
	if h.Accounts == nil {
		return nil, fmt.Errorf("account repository is not configured")
	}
	return h.Accounts.GetByID(id)
}

func (h *Handler) updateOperatorAccountRole(id uuid.UUID, roleID uuid.UUID) error {
	if h.UpdateAccountRole != nil {
		return h.UpdateAccountRole(id, roleID)
	}
	if h.Accounts == nil {
		return fmt.Errorf("account repository is not configured")
	}
	return h.Accounts.UpdateOperatorRole(id, roleID)
}

func (h *Handler) disableOperatorAccount(id uuid.UUID) error {
	if h.DisableAccount != nil {
		return h.DisableAccount(id)
	}
	if h.Accounts == nil {
		return fmt.Errorf("account repository is not configured")
	}
	now := time.Now().UTC()
	return h.Accounts.SetDisabledAt(id, &now)
}

func (h *Handler) countEnabledSuperadmins() (int64, error) {
	if h.CountEnabledSuperadmins != nil {
		return h.CountEnabledSuperadmins()
	}
	if h.Accounts == nil {
		return 0, fmt.Errorf("account repository is not configured")
	}
	return h.Accounts.CountEnabledSuperadmins(context.Background())
}

func (h *Handler) roleExists() operator.RoleExistsFunc {
	return operatorRoleExists(h.RoleExists, h.OperatorRoles)
}

func operatorRoleExists(override operator.RoleExistsFunc, repo *operator.Repository) operator.RoleExistsFunc {
	if override != nil {
		return override
	}
	if repo == nil {
		return nil
	}
	return func(id uuid.UUID) (bool, error) {
		_, err := repo.GetOperatorRoleByID(context.Background(), id)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}
}

func (h *Handler) loadMutableOperatorRole(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator role ID"})
		return uuid.Nil, false
	}
	if h.GetOperatorRole == nil && h.OperatorRoles == nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to load operator role"})
		return uuid.Nil, false
	}
	role, err := h.operatorRoleByID(c.Request.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Operator role not found"})
		return uuid.Nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to load operator role"})
		return uuid.Nil, false
	}
	if role.IsSystem {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: operator.ErrSystemRoleImmutable.Error()})
		return uuid.Nil, false
	}
	return role.ID, true
}

func (h *Handler) operatorRoleByID(ctx context.Context, id uuid.UUID) (sqlcgen.OperatorRole, error) {
	if h.GetOperatorRole != nil {
		return h.GetOperatorRole(ctx, id)
	}
	if h.OperatorRoles == nil {
		return sqlcgen.OperatorRole{}, fmt.Errorf("operator roles are not configured")
	}
	return h.OperatorRoles.GetOperatorRoleByID(ctx, id)
}

func parsePostedCustomGrants(c *gin.Context, keys []string) ([]operator.Permission, bool) {
	grants, err := operator.ParseCustomGrants(keys)
	if errors.Is(err, operator.ErrAdminIAMOnCustomRole) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: operator.ErrAdminIAMOnCustomRole.Error()})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator grant"})
		return nil, false
	}
	return grants, true
}

func grantKeys(grants []operator.Permission) []string {
	keys := make([]string, len(grants))
	for i, g := range grants {
		keys[i] = g.Key()
	}
	return keys
}

func writeOperatorRoleWriteError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, operator.ErrRoleNameTaken):
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, operator.ErrRoleAssigned):
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, operator.ErrRoleReferenced):
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, operator.ErrSystemRoleImmutable):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, operator.ErrReservedSystemRoleName):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, operator.ErrAdminIAMOnCustomRole):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, pgx.ErrNoRows):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Operator role not found"})
	default:
		log.Printf("operator role write: %v", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: fallback})
	}
}
