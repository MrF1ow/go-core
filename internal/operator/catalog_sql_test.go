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
	permissionInsertPattern         = regexp.MustCompile(`(?is)INSERT\s+INTO\s+operator_permissions\s*\([^)]*\)\s*VALUES\s*(.*?)\s*ON\s+CONFLICT`)
	permissionRowPattern            = regexp.MustCompile(`(?is)\(\s*'[^']+'\s*,\s*'([^']+)'\s*,\s*'([^']+)'\s*,\s*'[^']*'\s*\)`)
	roleGrantPattern                = regexp.MustCompile(`(?is)INSERT\s+INTO\s+operator_role_permissions\s*\([^)]*\)\s*SELECT\s+'([^']+)'\s*,\s*id\s+FROM\s+operator_permissions\s*(.*?)\s*ON\s+CONFLICT`)
	excludedResourcePattern         = regexp.MustCompile(`(?is)^WHERE\s+resource\s+<>\s+'([^']+)'\s*$`)
	permissionTuplePattern          = regexp.MustCompile(`(?is)\(\s*'([^']+)'\s*,\s*'([^']+)'\s*\)`)
	adminAccountRoleBackfillPattern = regexp.MustCompile(`(?is)UPDATE\s+admin_accounts\s+SET\s+operator_role_id\s+=\s+'([^']+)'`)
	adminKeyRoleBackfillPattern     = regexp.MustCompile(`(?is)UPDATE\s+api_keys\s+SET\s+operator_role_id\s+=\s+'([^']+)'\s+WHERE\s+key_type\s+=\s+'admin'\s+AND\s+operator_role_id\s+IS\s+NULL`)
)

func TestAdminAccountRoleBackfillUsesSuperadminID(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/017_admin_account_operator_role.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkAdminAccountRoleBackfill(string(sql), RoleIDSuperadmin); err != nil {
		t.Fatal(err)
	}
}

func TestAdminKeyOperatorRoleRequiredMigration(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/019_admin_key_operator_role_required.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	if !strings.Contains(text, "api_keys_admin_operator_role_required") {
		t.Fatal("019 missing CHECK name")
	}
	if !strings.Contains(text, "CHECK (key_type <> 'admin' OR operator_role_id IS NOT NULL)") {
		t.Fatal("019 missing admin-key CHECK")
	}
	match := adminKeyRoleBackfillPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		t.Fatal("019 missing viewer backfill")
	}
	got, err := uuid.Parse(match[1])
	if err != nil {
		t.Fatal(err)
	}
	if got != RoleIDViewer {
		t.Fatalf("backfill role ID = %s, want %s", got, RoleIDViewer)
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(schema), "api_keys_admin_operator_role_required") {
		t.Fatal("schema.sql missing admin-key CHECK")
	}
}

func TestAdminKeyMustExpireMigration(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/021_admin_key_must_expire.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	if !strings.Contains(text, "NOW() + INTERVAL '365 days'") {
		t.Fatal("021 missing 365-day backfill")
	}
	if !strings.Contains(text, "api_keys_admin_must_expire") {
		t.Fatal("021 missing CHECK name")
	}
	if !strings.Contains(text, "CHECK (key_type <> 'admin' OR expires_at IS NOT NULL)") {
		t.Fatal("021 missing admin-key expiry CHECK")
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(schema), "api_keys_admin_must_expire") {
		t.Fatal("schema.sql missing admin-key expiry CHECK")
	}
}

func TestAdminAccountAppMigration(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/022_admin_account_app.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	if !strings.Contains(text, "admin_accounts_superadmin_is_platform") {
		t.Fatal("022 missing superadmin CHECK name")
	}
	if !strings.Contains(text, "api_keys_admin_app_id_null") {
		t.Fatal("022 missing admin-key app_id CHECK name")
	}
	superadminCheck := "CHECK (operator_role_id <> '" + RoleIDSuperadmin.String() + "'::uuid OR app_id IS NULL)"
	if !strings.Contains(text, superadminCheck) {
		t.Fatal("022 missing superadmin platform CHECK")
	}
	if !strings.Contains(text, "CHECK (key_type <> 'admin' OR app_id IS NULL)") {
		t.Fatal("022 missing admin-key app_id CHECK")
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	schemaText := string(schema)
	if !strings.Contains(schemaText, "admin_accounts_superadmin_is_platform") {
		t.Fatal("schema.sql missing superadmin platform CHECK")
	}
	if !strings.Contains(schemaText, superadminCheck) {
		t.Fatal("schema.sql missing superadmin platform CHECK expression")
	}
	if !strings.Contains(schemaText, "api_keys_admin_app_id_null") {
		t.Fatal("schema.sql missing admin-key app_id CHECK")
	}
	countSQL, err := os.ReadFile("../queries/admin_account.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(countSQL), "app_id IS NULL") {
		t.Fatal("CountEnabledSuperadminAccounts missing app_id IS NULL")
	}
}

func TestOperatorOneWayRevokeMigration(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/023_operator_one_way_revoke.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	for _, needle := range []string{
		"admin_accounts_disabled_at_one_way",
		"api_keys_is_revoked_one_way",
		"disabled_at cannot be cleared",
		"is_revoked cannot be cleared",
		"EXECUTE FUNCTION prevent_admin_account_reenable",
		"EXECUTE FUNCTION prevent_api_key_unrevoke",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("023 missing %q", needle)
		}
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	schemaText := string(schema)
	for _, needle := range []string{
		"admin_accounts_disabled_at_one_way",
		"api_keys_is_revoked_one_way",
		"api_keys_admin_app_id_null",
		"disabled_at cannot be cleared",
		"is_revoked cannot be cleared",
	} {
		if !strings.Contains(schemaText, needle) {
			t.Fatalf("schema.sql missing %q", needle)
		}
	}
}

func TestOperatorIAMEvidenceMigrationExists(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/018_operator_iam_evidence.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	for _, needle := range []string{
		"operator_iam_events",
		"operator_access_logs",
		"disabled_at",
		"disable_principal",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("018 missing %q", needle)
		}
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	schemaText := string(schema)
	for _, needle := range []string{"operator_iam_events", "operator_access_logs", "disabled_at"} {
		if !strings.Contains(schemaText, needle) {
			t.Fatalf("schema.sql missing %q", needle)
		}
	}
}

func TestOperatorAccessLogClientMigrationExists(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/020_operator_access_log_client.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	for _, needle := range []string{"operator_access_logs", "ip_address", "user_agent"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("020 missing %q", needle)
		}
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	schemaText := string(schema)
	start := strings.Index(schemaText, "CREATE TABLE operator_access_logs")
	if start < 0 {
		t.Fatal("schema.sql missing operator_access_logs")
	}
	end := strings.Index(schemaText[start:], ";")
	if end < 0 {
		t.Fatal("schema.sql operator_access_logs unterminated")
	}
	block := schemaText[start : start+end]
	for _, needle := range []string{"ip_address", "user_agent"} {
		if !strings.Contains(block, needle) {
			t.Fatalf("schema.sql operator_access_logs missing %q", needle)
		}
	}
}

func checkAdminAccountRoleBackfill(sqlText string, want uuid.UUID) error {
	match := adminAccountRoleBackfillPattern.FindStringSubmatch(sqlText)
	if len(match) != 2 {
		return fmt.Errorf("admin_accounts operator_role_id backfill not found")
	}
	got, err := uuid.Parse(match[1])
	if err != nil {
		return fmt.Errorf("parse backfill role ID %q: %w", match[1], err)
	}
	if got != want {
		return fmt.Errorf("backfill role ID = %s, want %s", got, want)
	}
	return nil
}

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
