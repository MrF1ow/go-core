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
	return web.AttachCan(data, h.BasePath, p.Allows)
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
	return p.Allows(resource, action)
}

type userDetailTemplate struct {
	*UserDetail
	CanWrite    bool
	CanSessions bool
}

func (h *GUIHandler) userDetailView(c *gin.Context, detail *UserDetail) userDetailTemplate {
	return userDetailTemplate{
		UserDetail:  detail,
		CanWrite:    h.principalCan(c, operator.ResUsers, operator.ActionWrite),
		CanSessions: h.principalCan(c, operator.ResSessions, operator.ActionRead),
	}
}

type tenantListData struct {
	Tenants    []TenantListItem
	Page       int
	TotalPages int
	Total      int64
	CanWrite   bool
}

func (h *GUIHandler) tenantListView(c *gin.Context, tenants []TenantListItem, page, totalPages int, total int64) tenantListData {
	return tenantListData{
		Tenants:    tenants,
		Page:       page,
		TotalPages: totalPages,
		Total:      total,
		CanWrite:   h.principalCan(c, operator.ResTenants, operator.ActionWrite),
	}
}
