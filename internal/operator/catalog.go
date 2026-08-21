package operator

const (
	ActionRead  = "read"
	ActionWrite = "write"
)

const (
	ResDashboard     = "dashboard"
	ResTenants       = "tenants"
	ResApplications  = "applications"
	ResOAuth         = "oauth"
	ResOIDC          = "oidc"
	ResSessionGroups = "session_groups"
	ResUsers         = "users"
	ResSessions      = "sessions"
	ResIPRules       = "ip_rules"
	ResEndUserRBAC   = "end_user_rbac"
	ResEmail         = "email"
	ResLogs          = "logs"
	ResAPIKeys       = "api_keys"
	ResWebhooks      = "webhooks"
	ResMonitoring    = "monitoring"
	ResSettings      = "settings"
	ResAdminIAM      = "admin_iam"
)

const (
	RoleSuperadmin = "superadmin"
	RoleAdmin      = "admin"
	RoleSupport    = "support"
	RoleViewer     = "viewer"
)

// Permission is one catalog row: a resource plus a closed action.
type Permission struct {
	Resource string
	Action   string
}

// Key is the grant token stored on a principal, "resource:action".
func (p Permission) Key() string {
	return p.Resource + ":" + p.Action
}

var catalog = []Permission{
	{ResDashboard, ActionRead},
	{ResTenants, ActionRead},
	{ResTenants, ActionWrite},
	{ResApplications, ActionRead},
	{ResApplications, ActionWrite},
	{ResOAuth, ActionRead},
	{ResOAuth, ActionWrite},
	{ResOIDC, ActionRead},
	{ResOIDC, ActionWrite},
	{ResSessionGroups, ActionRead},
	{ResSessionGroups, ActionWrite},
	{ResUsers, ActionRead},
	{ResUsers, ActionWrite},
	{ResSessions, ActionRead},
	{ResSessions, ActionWrite},
	{ResIPRules, ActionRead},
	{ResIPRules, ActionWrite},
	{ResEndUserRBAC, ActionRead},
	{ResEndUserRBAC, ActionWrite},
	{ResEmail, ActionRead},
	{ResEmail, ActionWrite},
	{ResLogs, ActionRead},
	{ResAPIKeys, ActionRead},
	{ResAPIKeys, ActionWrite},
	{ResWebhooks, ActionRead},
	{ResWebhooks, ActionWrite},
	{ResMonitoring, ActionRead},
	{ResSettings, ActionRead},
	{ResSettings, ActionWrite},
	{ResAdminIAM, ActionRead},
	{ResAdminIAM, ActionWrite},
}

// Catalog is the frozen operator permission set. Seed SQL must match this list.
func Catalog() []Permission {
	out := make([]Permission, len(catalog))
	copy(out, catalog)
	return out
}

func allGrantKeys() []string {
	keys := make([]string, len(catalog))
	for i, p := range catalog {
		keys[i] = p.Key()
	}
	return keys
}

func keysExceptAdminIAM() []string {
	out := make([]string, 0, len(catalog)-2)
	for _, p := range catalog {
		if p.Resource == ResAdminIAM {
			continue
		}
		out = append(out, p.Key())
	}
	return out
}

// GrantsFor returns the closed grant set for a seeded role name.
// Unknown names return nil so a caller cannot treat garbage as viewer.
func GrantsFor(role string) []string {
	switch role {
	case RoleSuperadmin:
		return allGrantKeys()
	case RoleAdmin:
		return keysExceptAdminIAM()
	case RoleSupport:
		return []string{
			ResDashboard + ":" + ActionRead,
			ResUsers + ":" + ActionRead,
			ResUsers + ":" + ActionWrite,
			ResSessions + ":" + ActionRead,
			ResSessions + ":" + ActionWrite,
			ResLogs + ":" + ActionRead,
		}
	case RoleViewer:
		return []string{
			ResDashboard + ":" + ActionRead,
			ResUsers + ":" + ActionRead,
			ResLogs + ":" + ActionRead,
			ResMonitoring + ":" + ActionRead,
		}
	default:
		return nil
	}
}

// DefaultRoleForNewPrincipal is the role assigned to new DB admin keys and new GUI accounts.
func DefaultRoleForNewPrincipal() string {
	return RoleViewer
}

// SystemRoleNames is the four frozen jobs. Custom roles are extra rows, not these names.
func SystemRoleNames() []string {
	return []string{RoleSuperadmin, RoleAdmin, RoleSupport, RoleViewer}
}
