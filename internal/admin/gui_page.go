package admin

import (
	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/web"
)

func (h *GUIHandler) page(c *gin.Context) web.TemplateData {
	data := web.TemplateData{
		Theme:         web.GetTheme(c),
		AdminUsername: getAdminUsername(c),
		AdminID:       getAdminID(c),
		CSRFToken:     getCSRFToken(c),
	}
	p, ok := guiPrincipal(c)
	if !ok {
		return data
	}
	return web.AttachCan(data, h.BasePath, p.Has)
}

func guiPrincipal(c *gin.Context) (*operator.Principal, bool) {
	val, ok := c.Get(web.OperatorPrincipalKey)
	if !ok {
		return nil, false
	}
	p, ok := val.(*operator.Principal)
	return p, ok && p != nil
}

func (h *GUIHandler) principalCan(c *gin.Context, resource, action string) bool {
	p, ok := guiPrincipal(c)
	if !ok {
		return false
	}
	return p.Has(resource, action)
}
