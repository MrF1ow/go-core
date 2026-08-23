package operator

// PlatformResource is true for platform catalog resources and for unknown names.
// Bound operators (AppID set) are denied those. App-scoped resources return false.
func PlatformResource(resource string) bool {
	switch resource {
	case ResDashboard, ResUsers, ResSessions, ResLogs, ResAPIKeys, ResOAuth, ResIPRules, ResWebhooks, ResEndUserRBAC:
		return false
	default:
		return true
	}
}
