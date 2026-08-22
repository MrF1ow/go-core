package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

func (h *GUIHandler) operatorIAMHTML(c *gin.Context, view operatorIAMView) {
	data := h.page(c)
	data.ActivePage = "operator-iam"
	data.Data = view
	c.HTML(http.StatusOK, "operator_iam", data)
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
	roster, err := h.rosterPageData()
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
	roster, err := h.rosterPageData()
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
	c.HTML(http.StatusOK, "operator_iam_events", operatorIAMView{Tab: operatorTabEvents, Events: entries})
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
	c.HTML(http.StatusOK, "operator_access_logs", operatorIAMView{Tab: operatorTabAccessLogs, Logs: entries})
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
