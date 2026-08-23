package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/pkg/models"
)

func boundAppID(c *gin.Context) *uuid.UUID {
	p, ok := guiPrincipal(c)
	if !ok {
		return nil
	}
	return p.AppID
}

func restrictAppQuery(c *gin.Context, requested string) string {
	if bound := boundAppID(c); bound != nil {
		return bound.String()
	}
	return requested
}

func foreignApp(c *gin.Context, resourceApp uuid.UUID) bool {
	bound := boundAppID(c)
	if bound == nil {
		return false
	}
	return *bound != resourceApp
}

func foreignAppID(c *gin.Context, raw string) bool {
	id, err := uuid.Parse(raw)
	if err != nil {
		return boundAppID(c) != nil
	}
	return foreignApp(c, id)
}

func abortGUINotFound(c *gin.Context, body string) {
	c.String(http.StatusNotFound, body)
}

func abortGUINotFoundPage(c *gin.Context, name string, obj any) {
	c.HTML(http.StatusNotFound, name, obj)
}

func apiKeyForeign(c *gin.Context, key *models.ApiKey) bool {
	if boundAppID(c) == nil {
		return false
	}
	if key == nil || key.KeyType != KeyTypeApp || key.AppID == nil {
		return true
	}
	return foreignApp(c, *key.AppID)
}

func filterAppsWithTenant(c *gin.Context, apps []AppWithTenant) []AppWithTenant {
	bound := boundAppID(c)
	if bound == nil {
		return apps
	}
	for _, a := range apps {
		if a.ID == *bound {
			return []AppWithTenant{a}
		}
	}
	return []AppWithTenant{{ID: *bound}}
}

func filterApplications(c *gin.Context, apps []models.Application) []models.Application {
	bound := boundAppID(c)
	if bound == nil {
		return apps
	}
	for _, a := range apps {
		if a.ID == *bound {
			return []models.Application{a}
		}
	}
	return []models.Application{{ID: *bound}}
}

func (h *GUIHandler) appsForGUI(c *gin.Context) ([]AppWithTenant, error) {
	if h.Repo == nil {
		if bound := boundAppID(c); bound != nil {
			return []AppWithTenant{{ID: *bound}}, nil
		}
		return nil, nil
	}
	apps, err := h.Repo.ListAllAppsWithTenantName()
	if err != nil {
		return nil, err
	}
	return filterAppsWithTenant(c, apps), nil
}

func (h *GUIHandler) rbacAppsForGUI(c *gin.Context) []models.Application {
	if h.RBACService == nil || h.RBACService.Repo == nil {
		if bound := boundAppID(c); bound != nil {
			return []models.Application{{ID: *bound}}
		}
		return nil
	}
	apps, err := h.RBACService.Repo.ListAllApps()
	if err != nil {
		if bound := boundAppID(c); bound != nil {
			return []models.Application{{ID: *bound}}
		}
		return nil
	}
	return filterApplications(c, apps)
}

func (h *GUIHandler) listUsers(page, pageSize int, appID, search string) ([]UserListItem, int64, error) {
	if h.ListUsers != nil {
		return h.ListUsers(page, pageSize, appID, search)
	}
	if h.Repo == nil {
		return nil, 0, nil
	}
	return h.Repo.ListUsersWithDetails(page, pageSize, appID, search)
}

func (h *GUIHandler) userDetailByID(id string) (*UserDetail, error) {
	var detail *UserDetail
	var err error
	if h.GetUserDetail != nil {
		detail, err = h.GetUserDetail(id)
	} else if h.Repo == nil {
		return nil, errNotFound
	} else {
		detail, err = h.Repo.GetUserDetailByID(id)
	}
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, errNotFound
	}
	return detail, nil
}

func (h *GUIHandler) apiKeyListFilter(c *gin.Context) (keyType, appID string) {
	keyType = c.Query("key_type")
	if bound := boundAppID(c); bound != nil {
		return KeyTypeApp, bound.String()
	}
	return keyType, ""
}
