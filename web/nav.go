package web

import "strings"

// NavSpec is one frozen sidebar row. Resource and action are catalog strings.
type NavSpec struct {
	Heading, Page, Path, Icon, Label, Resource, Action string
}

// NavGroup is a heading plus the items the current principal may see.
type NavGroup struct {
	Heading string
	Items   []NavItem
}

// NavItem is one sidebar link after filtering.
type NavItem struct {
	Page, Path, Icon, Label string
}

var navSpec = []NavSpec{
	{Page: "dashboard", Path: "/", Icon: "bi-speedometer2", Label: "Dashboard", Resource: "dashboard", Action: "read"},

	{Heading: "Management", Page: "tenants", Path: "/tenants", Icon: "bi-building", Label: "Tenants", Resource: "tenants", Action: "read"},
	{Heading: "Management", Page: "applications", Path: "/applications", Icon: "bi-app-indicator", Label: "Applications", Resource: "applications", Action: "read"},
	{Heading: "Management", Page: "users", Path: "/users", Icon: "bi-people", Label: "Users", Resource: "users", Action: "read"},
	{Heading: "Management", Page: "oauth", Path: "/oauth", Icon: "bi-key", Label: "OAuth Config", Resource: "oauth", Action: "read"},
	{Heading: "Management", Page: "oidc-clients", Path: "/oidc-clients", Icon: "bi-fingerprint", Label: "OIDC Clients", Resource: "oidc", Action: "read"},
	{Heading: "Management", Page: "session-groups", Path: "/session-groups", Icon: "bi-link-45deg", Label: "Session Groups", Resource: "session_groups", Action: "read"},

	{Heading: "Security", Page: "sessions", Path: "/sessions", Icon: "bi-broadcast", Label: "Sessions", Resource: "sessions", Action: "read"},
	{Heading: "Security", Page: "ip-rules", Path: "/ip-rules", Icon: "bi-shield-lock", Label: "IP Rules", Resource: "ip_rules", Action: "read"},
	{Heading: "Security", Page: "roles", Path: "/roles", Icon: "bi-shield-check", Label: "Roles", Resource: "end_user_rbac", Action: "read"},
	{Heading: "Security", Page: "permissions", Path: "/permissions", Icon: "bi-lock", Label: "Permissions", Resource: "end_user_rbac", Action: "read"},
	{Heading: "Security", Page: "user-roles", Path: "/user-roles", Icon: "bi-person-badge", Label: "User Roles", Resource: "end_user_rbac", Action: "read"},

	{Heading: "Email", Page: "email-servers", Path: "/email-servers", Icon: "bi-hdd-network", Label: "Email Servers", Resource: "email", Action: "read"},
	{Heading: "Email", Page: "email-templates", Path: "/email-templates", Icon: "bi-envelope-paper", Label: "Email Templates", Resource: "email", Action: "read"},
	{Heading: "Email", Page: "email-types", Path: "/email-types", Icon: "bi-tags", Label: "Email Types", Resource: "email", Action: "read"},

	{Heading: "System", Page: "logs", Path: "/logs", Icon: "bi-journal-text", Label: "Activity Logs", Resource: "logs", Action: "read"},
	{Heading: "System", Page: "api-keys", Path: "/api-keys", Icon: "bi-key-fill", Label: "API Keys", Resource: "api_keys", Action: "read"},
	{Heading: "System", Page: "webhooks", Path: "/webhooks", Icon: "bi-broadcast", Label: "Webhooks", Resource: "webhooks", Action: "read"},
	{Heading: "System", Page: "monitoring", Path: "/monitoring", Icon: "bi-heart-pulse", Label: "System Health", Resource: "monitoring", Action: "read"},
	{Heading: "System", Page: "settings", Path: "/settings", Icon: "bi-gear", Label: "Settings", Resource: "settings", Action: "read"},
}

func buildNav(basePath string, can func(string, string) bool) []NavGroup {
	if can == nil {
		return nil
	}
	var groups []NavGroup
	var current NavGroup
	flush := func() {
		if len(current.Items) == 0 {
			current = NavGroup{}
			return
		}
		groups = append(groups, current)
		current = NavGroup{}
	}
	for _, spec := range navSpec {
		if spec.Heading != current.Heading {
			flush()
			current.Heading = spec.Heading
		}
		if !can(spec.Resource, spec.Action) {
			continue
		}
		current.Items = append(current.Items, NavItem{
			Page:  spec.Page,
			Path:  joinGUIPath(basePath, spec.Path),
			Icon:  spec.Icon,
			Label: spec.Label,
		})
	}
	flush()
	return groups
}

func joinGUIPath(basePath, path string) string {
	base := strings.TrimRight(basePath, "/")
	if path == "" || path == "/" {
		return base + "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}
