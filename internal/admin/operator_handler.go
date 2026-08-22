package admin

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/dto"
	"github.com/MrF1ow/go-core/pkg/models"
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
	if truncated {
		c.Header("X-Export-Truncated", "true")
	} else {
		c.Header("X-Export-Truncated", "false")
	}
	c.Header("Content-Disposition", `attachment; filename="operator-roster.csv"`)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Status(http.StatusOK)
	writer := csv.NewWriter(c.Writer)
	if err := writer.Write([]string{"kind", "id", "display_name", "role", "created_at", "last_used_at", "expires_at", "revoked"}); err != nil {
		log.Printf("operator roster csv header: %v", err)
		return
	}
	for _, entry := range entries {
		if err := writer.Write(rosterCSVRow(entry)); err != nil {
			log.Printf("operator roster csv row: %v", err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("operator roster csv flush: %v", err)
	}
}

func (h *Handler) loadRoster() ([]operator.RosterEntry, bool, error) {
	keys, err := h.rosterKeys()
	if err != nil {
		return nil, false, err
	}
	accounts, err := h.rosterAccounts()
	if err != nil {
		return nil, false, err
	}
	raw := 1 + len(keys) + len(accounts)
	return operator.BuildRoster(operator.EnvKeyRosterEntry(), keys, accounts), raw > operator.ExportMaxRows, nil
}

func (h *Handler) rosterKeys() ([]operator.RosterEntry, error) {
	if h.RosterKeys != nil {
		return h.RosterKeys()
	}
	if h.Repo == nil {
		return nil, fmt.Errorf("api key repository is not configured")
	}
	items, _, err := h.Repo.ListApiKeys(1, operator.ExportMaxRows, KeyTypeAdmin)
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
	if h.RosterAccounts != nil {
		return h.RosterAccounts()
	}
	if h.Accounts == nil {
		return nil, fmt.Errorf("account repository is not configured")
	}
	accounts, err := h.Accounts.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]operator.RosterEntry, 0, len(accounts))
	for i := range accounts {
		account := accounts[i]
		id := account.ID
		out = append(out, operator.RosterEntry{
			Kind:        string(operator.KindGUIAccount),
			DisplayName: account.Username,
			RoleName:    h.accountRoleName(account),
			AccountID:   &id,
			CreatedAt:   account.CreatedAt,
			LastUsedAt:  account.LastLoginAt,
		})
	}
	return out, nil
}

func (h *Handler) accountRoleName(account models.AdminAccount) string {
	if name, ok := operator.RoleNameForID(account.OperatorRoleID); ok {
		return name
	}
	if h.OperatorRoles == nil {
		return ""
	}
	name, err := h.OperatorRoles.RoleName(context.Background(), account.OperatorRoleID)
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
	return []string{
		entry.Kind,
		id,
		entry.DisplayName,
		entry.RoleName,
		csvTime(entry.CreatedAt),
		csvTimePtr(entry.LastUsedAt),
		csvTimePtr(entry.ExpiresAt),
		fmt.Sprintf("%t", entry.Revoked),
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
