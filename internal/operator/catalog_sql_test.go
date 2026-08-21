package operator

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var (
	permissionInsertPattern = regexp.MustCompile(`(?is)INSERT\s+INTO\s+operator_permissions\s*\([^)]*\)\s*VALUES\s*(.*?)\s*ON\s+CONFLICT`)
	permissionRowPattern    = regexp.MustCompile(`(?is)\(\s*'[^']+'\s*,\s*'([^']+)'\s*,\s*'([^']+)'\s*,\s*'[^']*'\s*\)`)
	roleGrantPattern        = regexp.MustCompile(`(?is)INSERT\s+INTO\s+operator_role_permissions\s*\([^)]*\)\s*SELECT\s+'([^']+)'\s*,\s*id\s+FROM\s+operator_permissions\s*(.*?)\s*ON\s+CONFLICT`)
	excludedResourcePattern = regexp.MustCompile(`(?is)^WHERE\s+resource\s+<>\s+'([^']+)'\s*$`)
	permissionTuplePattern  = regexp.MustCompile(`(?is)\(\s*'([^']+)'\s*,\s*'([^']+)'\s*\)`)
)

func TestOperatorSeedMatchesCatalogAndGrants(t *testing.T) {
	seed, err := os.ReadFile("../../migrations/016_operator_rbac.sql")
	if err != nil {
		t.Fatal(err)
	}

	expectedGrants := map[uuid.UUID][]string{
		RoleIDSuperadmin: GrantsFor(RoleSuperadmin),
		RoleIDAdmin:      GrantsFor(RoleAdmin),
		RoleIDSupport:    GrantsFor(RoleSupport),
		RoleIDViewer:     GrantsFor(RoleViewer),
	}
	if err := validateOperatorSeedSQL(string(seed), Catalog(), expectedGrants); err != nil {
		t.Fatal(err)
	}
}

func TestValidateOperatorSeedSQLDetectsMissingPermissionInsert(t *testing.T) {
	const fixture = `
INSERT INTO operator_permissions (id, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000001', 'dashboard', 'read', 'View dashboard'),
    ('c0000000-0000-0000-0000-000000000002', 'tenants', 'read', 'View tenants')
ON CONFLICT (resource, action) DO NOTHING;

INSERT INTO operator_role_permissions (role_id, permission_id)
SELECT 'd0000000-0000-0000-0000-000000000001', id FROM operator_permissions
ON CONFLICT DO NOTHING;
`
	catalog := []Permission{
		{Resource: ResDashboard, Action: ActionRead},
		{Resource: ResTenants, Action: ActionRead},
	}
	grants := map[uuid.UUID][]string{
		RoleIDSuperadmin: {ResDashboard + ":" + ActionRead, ResTenants + ":" + ActionRead},
	}
	missing := strings.Replace(
		fixture,
		"\n    ('c0000000-0000-0000-0000-000000000002', 'tenants', 'read', 'View tenants')",
		"",
		1,
	)

	if err := validateOperatorSeedSQL(missing, catalog, grants); err == nil {
		t.Fatal("validation passed after a permission INSERT row was removed")
	}
}

func validateOperatorSeedSQL(sqlText string, expectedCatalog []Permission, expectedGrants map[uuid.UUID][]string) error {
	catalog, grants, err := parseOperatorSeedSQL(sqlText)
	if err != nil {
		return err
	}

	wantCatalog := make(map[string]struct{}, len(expectedCatalog))
	for _, permission := range expectedCatalog {
		wantCatalog[permission.Key()] = struct{}{}
	}
	if err := compareKeySets("permission catalog", catalog, wantCatalog); err != nil {
		return err
	}
	if len(grants) != len(expectedGrants) {
		return fmt.Errorf("seeded role count = %d, want %d", len(grants), len(expectedGrants))
	}
	for roleID, keys := range expectedGrants {
		want := make(map[string]struct{}, len(keys))
		for _, key := range keys {
			want[key] = struct{}{}
		}
		got, ok := grants[roleID]
		if !ok {
			return fmt.Errorf("missing grants for role %s", roleID)
		}
		if err := compareKeySets("grants for "+roleID.String(), got, want); err != nil {
			return err
		}
	}
	return nil
}

func parseOperatorSeedSQL(sqlText string) (map[string]struct{}, map[uuid.UUID]map[string]struct{}, error) {
	permissionInsert := permissionInsertPattern.FindStringSubmatch(sqlText)
	if len(permissionInsert) != 2 {
		return nil, nil, fmt.Errorf("operator permission INSERT not found")
	}
	rows := permissionRowPattern.FindAllStringSubmatch(permissionInsert[1], -1)
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("operator permission INSERT has no rows")
	}
	catalog := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		catalog[row[1]+":"+row[2]] = struct{}{}
	}

	grantBlocks := roleGrantPattern.FindAllStringSubmatch(sqlText, -1)
	if len(grantBlocks) == 0 {
		return nil, nil, fmt.Errorf("operator role grant INSERTs not found")
	}
	grants := make(map[uuid.UUID]map[string]struct{}, len(grantBlocks))
	for _, block := range grantBlocks {
		roleID, err := uuid.Parse(block[1])
		if err != nil {
			return nil, nil, fmt.Errorf("parse role ID %q: %w", block[1], err)
		}
		keys, err := grantsForSQLWhere(catalog, strings.TrimSpace(block[2]))
		if err != nil {
			return nil, nil, fmt.Errorf("parse grants for role %s: %w", roleID, err)
		}
		grants[roleID] = keys
	}
	return catalog, grants, nil
}

func grantsForSQLWhere(catalog map[string]struct{}, where string) (map[string]struct{}, error) {
	if where == "" {
		return cloneKeySet(catalog), nil
	}
	if match := excludedResourcePattern.FindStringSubmatch(where); len(match) == 2 {
		out := make(map[string]struct{}, len(catalog))
		prefix := match[1] + ":"
		for key := range catalog {
			if !strings.HasPrefix(key, prefix) {
				out[key] = struct{}{}
			}
		}
		return out, nil
	}
	if strings.HasPrefix(strings.ToUpper(where), "WHERE (RESOURCE, ACTION) IN") {
		tuples := permissionTuplePattern.FindAllStringSubmatch(where, -1)
		if len(tuples) == 0 {
			return nil, fmt.Errorf("permission tuple list is empty")
		}
		out := make(map[string]struct{}, len(tuples))
		for _, tuple := range tuples {
			key := tuple[1] + ":" + tuple[2]
			if _, ok := catalog[key]; !ok {
				return nil, fmt.Errorf("grant references unseeded permission %q", key)
			}
			out[key] = struct{}{}
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported grant predicate %q", where)
}

func compareKeySets(label string, got, want map[string]struct{}) error {
	if len(got) == len(want) {
		equal := true
		for key := range want {
			if _, ok := got[key]; !ok {
				equal = false
				break
			}
		}
		if equal {
			return nil
		}
	}
	return fmt.Errorf("%s mismatch: got %v, want %v", label, sortedKeys(got), sortedKeys(want))
}

func cloneKeySet(source map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(source))
	for key := range source {
		out[key] = struct{}{}
	}
	return out
}

func sortedKeys(keys map[string]struct{}) []string {
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
