package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/web"
)

const (
	guiPageContentID    = "page-content"
	guiForbiddenMessage = "You do not have permission to view this page."
	guiAuthBugMessage   = "internal authentication error"
)

var htmxTargetIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,127}$`)

// RequireGUIPermission is the HTML sibling of RequireOperatorPermission.
// Missing principal is 500 HTML. Deny is AbortGUIForbidden. Never JSON.
func RequireGUIPermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := principalFromContext(c)
		if !ok {
			AbortGUIInternal(c)
			return
		}
		if !p.Has(resource, action) {
			AbortGUIForbidden(c)
			return
		}
		c.Next()
	}
}

// AbortGUIForbidden writes HTTP 403 HTML and aborts.
// Typed URLs and #page-content HTMX get a page-shaped body.
// Other HTMX targets get a fragment-shaped body whose outer id matches the target.
func AbortGUIForbidden(c *gin.Context) {
	c.Header(web.GUIForbiddenHeader, web.GUIForbiddenValue)
	c.Header("Content-Type", "text/html; charset=utf-8")
	targetID := guiPageContentID
	if c.GetHeader("HX-Request") == "true" {
		if id, ok := sanitizeHTMXTarget(c.GetHeader("HX-Target")); ok && id != "" && id != guiPageContentID {
			targetID = id
		}
	}
	c.String(http.StatusForbidden, `<div id="`+targetID+`">`+guiForbiddenMessage+`</div>`)
	c.Abort()
}

// AbortGUIInternal writes HTTP 500 HTML for a missing principal on a cookie route.
func AbortGUIInternal(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusInternalServerError, guiAuthBugMessage)
	c.Abort()
}

func principalFromContext(c *gin.Context) (*operator.Principal, bool) {
	val, ok := c.Get(web.OperatorPrincipalKey)
	if !ok {
		return nil, false
	}
	p, ok := val.(*operator.Principal)
	return p, ok && p != nil
}

func sanitizeHTMXTarget(raw string) (string, bool) {
	id := strings.TrimPrefix(strings.TrimSpace(raw), "#")
	if !htmxTargetIDPattern.MatchString(id) {
		return "", false
	}
	return id, true
}
