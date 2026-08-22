package admin

import (
	"context"
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
	"github.com/MrF1ow/go-core/pkg/dto"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

type accessLogResponse struct {
	Entries []operator.AccessRecord `json:"entries"`
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
	limit := int32(100)
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = int32(n)
		}
	}

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

func (h *Handler) listAccessLogs(ctx context.Context, limit int32, decision *string) ([]operator.AccessRecord, error) {
	if h.AccessLogList != nil {
		return h.AccessLogList(ctx, limit, decision)
	}
	if h.OperatorRoles == nil {
		return nil, fmt.Errorf("operator repository is not configured")
	}
	return h.OperatorRoles.ListAccessLogs(ctx, limit, 0, decision)
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
	limit := int32(100)
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = int32(n)
		}
	}

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

func (h *Handler) listIAMEvents(ctx context.Context, limit int32, targetKeyID, targetAccountID *uuid.UUID) ([]operator.IAMEvent, error) {
	if h.IAMEventList != nil {
		return h.IAMEventList(ctx, limit, targetKeyID, targetAccountID)
	}
	if h.OperatorRoles == nil {
		return nil, fmt.Errorf("operator repository is not configured")
	}
	return h.OperatorRoles.ListIAMEvents(ctx, limit, 0, targetKeyID, targetAccountID)
}

type assignKeyRoleRequest struct {
	OperatorRoleID string `json:"operator_role_id"`
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
	roleID, err := operator.ParseAssignedAdminRole(*p, req.OperatorRoleID, key.KeyType, key.OperatorRoleID)
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
	Password string `json:"password" binding:"required,min=12"`
}

type operatorAccountResponse struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	OperatorRoleID uuid.UUID `json:"operator_role_id"`
	Role           string    `json:"role"`
}

type assignAccountRoleRequest struct {
	OperatorRoleID string `json:"operator_role_id"`
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
		account.ID = uuid.New()
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

// OperatorAccountRole assigns a system operator role to a GUI account.
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
	account, ok := h.loadOperatorAccountOrAbort(c, id)
	if !ok {
		return
	}
	roleID, err := uuid.Parse(strings.TrimSpace(req.OperatorRoleID))
	if err != nil || !operator.IsSystemRoleID(roleID) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid operator role"})
		return
	}
	if account.OperatorRoleID == roleID {
		c.Status(http.StatusNoContent)
		return
	}
	if roleID != operator.RoleIDSuperadmin {
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
	if err := h.updateOperatorAccountRole(account.ID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to update operator account"})
		return
	}
	oldRole := account.OperatorRoleID
	newRole := roleID
	accountID := account.ID
	h.writeJSONIAMEvent(c, operator.IAMEvent{
		TargetKind:      operator.KindGUIAccount,
		TargetAccountID: &accountID,
		OldRoleID:       &oldRole,
		NewRoleID:       &newRole,
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
