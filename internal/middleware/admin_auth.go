package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/JedidiahDigital/go-core/internal/admin"
	"github.com/JedidiahDigital/go-core/web"
)

// AdminAuthMiddleware validates the Admin API Key header.
// It checks the static ADMIN_API_KEY env var first (fast path, backward compatible),
// then falls back to looking up hashed admin-type keys in the database.
// If keyValidator is nil, only the static env var is checked.
func AdminAuthMiddleware(adminAPIKey string, keyValidator web.ApiKeyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Admin-API-Key")

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-Admin-API-Key header is required"})
			return
		}

		// Fast path: check static key with timing-safe comparison
		if adminAPIKey != "" {
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(adminAPIKey)) == 1 {
				c.Set(web.AuthTypeKey, web.AuthTypeAdmin)
				c.Next()
				return
			}
		}

		// Fallback: check DB-backed admin keys by SHA-256 hash
		if keyValidator != nil {
			h := sha256.Sum256([]byte(apiKey))
			keyHash := hex.EncodeToString(h[:])

			foundKey, err := keyValidator.FindActiveKeyByHash(keyHash)
			if err == nil && foundKey != nil && foundKey.KeyType == admin.KeyTypeAdmin {
				// Update last_used_at and increment daily usage asynchronously
				go keyValidator.UpdateApiKeyLastUsed(foundKey.ID)
				go keyValidator.IncrementDailyUsage(foundKey.ID)
				scopes := parseScopes(foundKey.Scopes)
				c.Set(web.ApiKeyScopesKey, scopes)
				c.Set(web.AuthTypeKey, web.AuthTypeAdmin)
				c.Next()
				return
			}
		}

		// If static key is not configured and no DB key matched, give a useful error
		if adminAPIKey == "" && keyValidator == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Admin API access not configured"})
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Admin API Key"})
	}
}
