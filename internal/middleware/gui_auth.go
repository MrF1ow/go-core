package middleware

import (
	"net/http"
	"strings"

	"github.com/MrF1ow/go-core/web"
	"github.com/gin-gonic/gin"
)

// GUIAuthMiddleware validates admin sessions via HTTP-only cookies.
// Unauthenticated requests are redirected to the login page.
// basePath is the URL prefix for the admin GUI (e.g. "/gui").
func GUIAuthMiddleware(sessionValidator web.SessionValidator, basePath string) gin.HandlerFunc {
	loginPath := basePath + "/login"
	twoFAPath := basePath + "/2fa-verify"
	twoFAResendPath := basePath + "/2fa-resend-email"
	passkeyBeginPath := basePath + "/passkey-login/begin"
	passkeyFinishPath := basePath + "/passkey-login/finish"
	staticPrefix := basePath + "/static/"
	magicLinkPath := basePath + "/magic-link-login"
	magicLinkVerifyPath := basePath + "/magic-link-login/verify"

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if path == loginPath ||
			path == twoFAPath ||
			path == twoFAResendPath ||
			path == passkeyBeginPath ||
			path == passkeyFinishPath ||
			path == magicLinkPath ||
			path == magicLinkVerifyPath ||
			strings.HasPrefix(path, staticPrefix) {
			c.Next()
			return
		}

		sessionID, err := c.Cookie(web.AdminSessionCookie)
		if err != nil || sessionID == "" {
			redirectToLogin(c, basePath)
			return
		}

		account, err := sessionValidator.ValidateSession(sessionID)
		if err != nil {
			web.ClearSessionCookie(c, basePath)
			redirectToLogin(c, basePath)
			return
		}

		c.Set(web.GUIAdminIDKey, account.ID.String())
		c.Set(web.GUIAdminUsernameKey, account.Username)
		c.Set(web.GUISessionIDKey, sessionID)

		c.Next()
	}
}

func redirectToLogin(c *gin.Context, basePath string) {
	originalURL := c.Request.URL.Path
	loginPath := basePath + "/login"
	if originalURL == basePath+"/" || originalURL == basePath {
		c.Redirect(http.StatusFound, loginPath)
	} else {
		c.Redirect(http.StatusFound, loginPath+"?redirect="+originalURL)
	}
	c.Abort()
}
