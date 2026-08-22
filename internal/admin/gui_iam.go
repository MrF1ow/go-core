package admin

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
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

type rosterView struct {
	Entries   []rosterViewEntry
	Truncated bool
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

func (h *GUIHandler) rosterPageData() (rosterView, error) {
	entries, truncated, err := h.loadRoster()
	if err != nil {
		return rosterView{}, err
	}
	rows := make([]rosterViewEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, rosterViewEntry{RosterEntry: entry, Status: rosterStatus(entry)})
	}
	return rosterView{Entries: rows, Truncated: truncated}, nil
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
	data := h.page(c)
	data.ActivePage = "operator-iam"
	data.Data = roster
	c.HTML(http.StatusOK, "operator_iam", data)
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
