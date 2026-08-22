package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/dto"
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
