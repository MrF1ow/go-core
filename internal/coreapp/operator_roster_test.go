package coreapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
)

type rosterKeyStore struct {
	keys map[string]*models.ApiKey
}

func (s *rosterKeyStore) FindActiveKeyByHash(keyHash string) (*models.ApiKey, error) {
	return s.keys[keyHash], nil
}

func (s *rosterKeyStore) UpdateApiKeyLastUsed(uuid.UUID) {}

func (s *rosterKeyStore) IncrementDailyUsage(uuid.UUID) {}

func (s *rosterKeyStore) put(raw string, key *models.ApiKey) {
	if s.keys == nil {
		s.keys = map[string]*models.ApiKey{}
	}
	sum := sha256.Sum256([]byte(raw))
	key.KeyHash = hex.EncodeToString(sum[:])
	s.keys[key.KeyHash] = key
}

type rosterGrants struct {
	byID map[uuid.UUID]struct {
		name string
		keys []string
	}
}

func (s *rosterGrants) RoleGrants(_ context.Context, roleID uuid.UUID) (string, []string, error) {
	g, ok := s.byID[roleID]
	if !ok {
		return "", nil, pgx.ErrNoRows
	}
	return g.name, g.keys, nil
}

func rosterTestEngine(t *testing.T, roleID uuid.UUID, roleName string, handler *admin.Handler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	raw := "roster-test-key"
	store := &rosterKeyStore{}
	id := uuid.New()
	rid := roleID
	store.put(raw, &models.ApiKey{
		ID:             id,
		KeyType:        admin.KeyTypeAdmin,
		OperatorRoleID: &rid,
	})
	grants := &rosterGrants{byID: map[uuid.UUID]struct {
		name string
		keys []string
	}{
		roleID: {name: roleName, keys: operator.GrantsFor(roleName)},
	}}
	engine := gin.New()
	group := engine.Group("/admin")
	group.Use(middleware.AdminAuthMiddleware("", store, grants))
	group.GET("/operator/roster", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorRoster)
	group.GET("/operator/roster/export", requireOp(operator.ResAdminIAM, operator.ActionRead), handler.OperatorRosterExport)
	return engine
}

func rosterGET(engine *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Admin-API-Key", "roster-test-key")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestOperatorRoster_ViewerForbidden(t *testing.T) {
	handler := &admin.Handler{
		RosterKeys:     func() ([]operator.RosterEntry, error) { return nil, nil },
		RosterAccounts: func() ([]operator.RosterEntry, error) { return nil, nil },
	}
	engine := rosterTestEngine(t, operator.RoleIDViewer, operator.RoleViewer, handler)
	rec := rosterGET(engine, "/admin/operator/roster")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestOperatorRoster_SuperadminIncludesEnvKeyAndRole(t *testing.T) {
	accountID := uuid.New()
	handler := &admin.Handler{
		RosterKeys: func() ([]operator.RosterEntry, error) {
			return []operator.RosterEntry{{
				Kind:        string(operator.KindAPIKey),
				DisplayName: "ops",
				RoleName:    operator.RoleAdmin,
			}}, nil
		},
		RosterAccounts: func() ([]operator.RosterEntry, error) {
			return []operator.RosterEntry{{
				Kind:        string(operator.KindGUIAccount),
				DisplayName: "ada",
				RoleName:    operator.RoleSuperadmin,
				AccountID:   &accountID,
				CreatedAt:   time.Now(),
			}}, nil
		},
	}
	engine := rosterTestEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin, handler)
	rec := rosterGET(engine, "/admin/operator/roster")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries []operator.RosterEntry `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) < 2 {
		t.Fatalf("entries = %#v", body.Entries)
	}
	if body.Entries[0].Kind != string(operator.KindEnvKey) || body.Entries[0].RoleName != operator.RoleSuperadmin {
		t.Fatalf("env = %+v", body.Entries[0])
	}
	found := false
	for _, entry := range body.Entries {
		if entry.RoleName == operator.RoleSuperadmin {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing superadmin role in %#v", body.Entries)
	}
}

func TestOperatorRoster_DisabledAccountJSON(t *testing.T) {
	accountID := uuid.New()
	disabled := true
	handler := &admin.Handler{
		RosterKeys: func() ([]operator.RosterEntry, error) { return nil, nil },
		RosterAccounts: func() ([]operator.RosterEntry, error) {
			return []operator.RosterEntry{{
				Kind:        string(operator.KindGUIAccount),
				DisplayName: "bob",
				RoleName:    operator.RoleViewer,
				AccountID:   &accountID,
				Disabled:    &disabled,
			}}, nil
		},
	}
	engine := rosterTestEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin, handler)
	rec := rosterGET(engine, "/admin/operator/roster")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Entries []operator.RosterEntry `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Entries[0].Disabled != nil {
		t.Fatalf("env disabled = %+v", body.Entries[0])
	}
	found := false
	for _, entry := range body.Entries {
		if entry.AccountID != nil && *entry.AccountID == accountID {
			found = true
			if entry.Disabled == nil || !*entry.Disabled {
				t.Fatalf("account = %+v", entry)
			}
		}
	}
	if !found {
		t.Fatalf("missing account in %#v", body.Entries)
	}
}

func TestOperatorRosterExport_CSVAndTruncationHeader(t *testing.T) {
	handler := &admin.Handler{
		RosterKeys:     func() ([]operator.RosterEntry, error) { return nil, nil },
		RosterAccounts: func() ([]operator.RosterEntry, error) { return nil, nil },
	}
	engine := rosterTestEngine(t, operator.RoleIDSuperadmin, operator.RoleSuperadmin, handler)
	rec := rosterGET(engine, "/admin/operator/roster/export")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Export-Truncated") != "false" {
		t.Fatalf("truncated = %q", rec.Header().Get("X-Export-Truncated"))
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "operator-roster.csv") {
		t.Fatalf("disposition = %q", rec.Header().Get("Content-Disposition"))
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "env_key") {
		t.Fatalf("csv = %s", body)
	}
}
