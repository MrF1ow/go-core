package admin

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
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
