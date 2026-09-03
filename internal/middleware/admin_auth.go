package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/MrF1ow/go-core/web"
)

// AdminAuthMiddleware validates the Admin API Key header.
// The env key is a synthetic superadmin principal. A DB admin key loads its
// operator role. Expired, revoked, unknown, and null-role admin keys are 401.
// If grants is nil, only the env key can proceed.
func AdminAuthMiddleware(adminAPIKey string, keyValidator web.ApiKeyValidator, grants operator.GrantLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Admin-API-Key")

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-Admin-API-Key header is required"})
			return
		}

		if adminAPIKey != "" {
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(adminAPIKey)) == 1 {
				p := operator.SuperadminPrincipal(operator.KindEnvKey)
				c.Set(web.OperatorPrincipalKey, &p)
				c.Set(web.AuthTypeKey, web.AuthTypeAdmin)
				c.Next()
				return
			}
		}

		if keyValidator != nil {
			h := sha256.Sum256([]byte(apiKey))
			keyHash := hex.EncodeToString(h[:])

			foundKey, err := keyValidator.FindActiveKeyByHash(keyHash)
			if err == nil && foundKey != nil && foundKey.KeyType == admin.KeyTypeAdmin {
				p, ok := principalForDBKey(c, foundKey, grants)
				if !ok {
					return
				}
				go keyValidator.UpdateApiKeyLastUsed(foundKey.ID)
				go keyValidator.IncrementDailyUsage(foundKey.ID)
				c.Set(web.OperatorPrincipalKey, p)
				c.Set(web.AuthTypeKey, web.AuthTypeAdmin)
				c.Next()
				return
			}
		}

		if adminAPIKey == "" && keyValidator == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Admin API access not configured"})
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Admin API Key"})
	}
}

func principalForDBKey(c *gin.Context, foundKey *models.ApiKey, grants operator.GrantLookup) (*operator.Principal, bool) {
	if foundKey.OperatorRoleID == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Admin API Key"})
		return nil, false
	}
	if grants == nil {
		log.Printf("operator grant lookup is not configured")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal authentication error"})
		return nil, false
	}
	name, keys, err := grants.RoleGrants(c.Request.Context(), *foundKey.OperatorRoleID)
	if err != nil {
		log.Printf("operator grant lookup failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal authentication error"})
		return nil, false
	}
	p := operator.NewPrincipal(operator.KindAPIKey, name, keys)
	id := foundKey.ID
	p.KeyID = &id
	if foundKey.AppID != nil {
		appID := *foundKey.AppID
		p.AppID = &appID
	}
	return &p, true
}

// RequireOperatorPermission allows the request when the attached principal has
// the exact resource:action grant. Missing principal is a server bug (500).
func RequireOperatorPermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Get(web.OperatorPrincipalKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal authentication error"})
			return
		}
		p, ok := val.(*operator.Principal)
		if !ok || p == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal authentication error"})
			return
		}
		if !p.Allows(resource, action) {
			maybeLogOperatorAccess(c, p, resource, action, operator.DecisionDeny, http.StatusForbidden)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
		maybeLogOperatorAccess(c, p, resource, action, operator.DecisionAllow, c.Writer.Status())
	}
}
