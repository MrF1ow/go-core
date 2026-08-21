package operator

import "github.com/google/uuid"

// Kind identifies the single principal on an admin request.
type Kind string

const (
	KindEnvKey     Kind = "env_key"
	KindAPIKey     Kind = "api_key"
	KindGUIAccount Kind = "gui_account"
)

// Principal is the grant attached after admin authentication.
// Perms are exact "resource:action" keys. Wildcards are not grants in v1.
type Principal struct {
	Kind      Kind
	KeyID     *uuid.UUID
	AccountID *uuid.UUID
	RoleName  string
	perms     map[string]struct{}
}

// NewPrincipal builds a principal from a closed list of grant keys.
func NewPrincipal(kind Kind, roleName string, grantKeys []string) Principal {
	perms := make(map[string]struct{}, len(grantKeys))
	for _, k := range grantKeys {
		if k == "" {
			continue
		}
		perms[k] = struct{}{}
	}
	return Principal{
		Kind:     kind,
		RoleName: roleName,
		perms:    perms,
	}
}

// SuperadminPrincipal is the in-memory env-key grant. No DB.
func SuperadminPrincipal(kind Kind) Principal {
	return NewPrincipal(kind, RoleSuperadmin, GrantsFor(RoleSuperadmin))
}

// Has reports an exact resource:action grant. Unknown pairs are deny.
func (p Principal) Has(resource, action string) bool {
	if p.perms == nil {
		return false
	}
	_, ok := p.perms[resource+":"+action]
	return ok
}
