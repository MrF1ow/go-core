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
	val, ok := c.Get(web.OperatorPrincipalKey)
	if !ok {
		return data
	}
	p, ok := val.(*operator.Principal)
	if !ok || p == nil {
		return data
	}
	return web.AttachCan(data, h.BasePath, p.Has)
}
