package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/geoip"
	"github.com/MrF1ow/go-core/internal/util"
	"github.com/MrF1ow/go-core/pkg/dto"
)

func IPRuleMiddleware(evaluate func(appID uuid.UUID, clientIP string) geoip.AccessResult) gin.HandlerFunc {
	return func(c *gin.Context) {
		if evaluate == nil {
			c.Next()
			return
		}

		appIDVal, exists := c.Get(AppIDKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "X-App-ID header is required"})
			return
		}
		appID, ok := appIDVal.(uuid.UUID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid App ID in context"})
			return
		}

		result := evaluate(appID, util.GetClientIP(c))
		if !result.Allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse{Error: "Access denied from your location"})
			return
		}

		c.Next()
	}
}
