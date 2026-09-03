package coreapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	core "github.com/MrF1ow/go-core"
	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/geoip"
	applog "github.com/MrF1ow/go-core/internal/log"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/pkg/models"
)

const bindLiveEnvKey = "bind-live-env-key"

func TestBindLive_Lanes(t *testing.T) {
	ctx := context.Background()
	pool := bindLivePool(t)
	t.Cleanup(pool.Close)
	if err := core.RunCoreMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}

	defaultApp := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	var otherApp uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO applications (tenant_id, name) VALUES ($1, 'other-bind-live') RETURNING id`, defaultApp).Scan(&otherApp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM applications WHERE id = $1`, otherApp)
	})

	homeUser := uuid.New()
	otherUser := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, app_id, email) VALUES ($1, $2, $3), ($4, $5, $6)`,
		homeUser, defaultApp, "home-"+homeUser.String()+"@example.com",
		otherUser, otherApp, "other-"+otherUser.String()+"@example.com",
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1, $2)`, homeUser, otherUser)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO activity_logs (app_id, user_id, event_type, timestamp, severity)
		VALUES ($1, $2, 'login_success', NOW(), 'INFORMATIONAL'),
		       ($3, $4, 'login_success', NOW(), 'INFORMATIONAL')
	`, defaultApp, homeUser, otherApp, otherUser); err != nil {
		t.Fatal(err)
	}

	adminRepo := admin.NewRepository(pool)
	opRepo := operator.NewRepository(pool)
	logHandler := applog.NewHandler(applog.NewQueryService(applog.NewRepository(pool)))
	handler := &admin.Handler{
		CreateAPIKey: adminRepo.CreateApiKey,
		GetAPIKey:    adminRepo.GetApiKeyByID,
		IPRuleRepo:   geoip.NewIPRuleRepository(pool),
		RoleExists: func(id uuid.UUID) (bool, error) {
			return operator.IsSystemRoleID(id), nil
		},
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/admin")
	group.Use(middleware.AdminAuthMiddleware(bindLiveEnvKey, adminRepo, opRepo))
	group.POST("/operator/keys", requireOp(operator.ResAPIKeys, operator.ActionWrite), handler.OperatorCreateKey)
	group.GET("/tenants", requireOp(operator.ResTenants, operator.ActionRead), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	group.GET("/apps/:id/ip-rules", requireOp(operator.ResIPRules, operator.ActionRead), handler.ListIPRules)
	group.GET("/activity-logs", requireOp(operator.ResLogs, operator.ActionRead), logHandler.GetAllActivityLogs)
	group.GET("/operator/keys/:id", requireOp(operator.ResAPIKeys, operator.ActionRead), func(c *gin.Context) {
		key, err := handler.GetAPIKey(c.Param("id"))
		if err != nil || key == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, key)
	})

	outDir := filepath.Join("/opt/cursor/artifacts")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("/tmp/swarm-bind", 0o755); err != nil {
		t.Fatal(err)
	}

	body := `{"name":"bound-viewer","operator_role_id":"` + operator.RoleIDViewer.String() + `","app_id":"` + defaultApp.String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	mint := bindLiveDo(engine, http.MethodPost, "/admin/operator/keys", bindLiveEnvKey, body)
	saveBindLive(t, outDir, "bind-mint-201.json", mint)
	if mint.Code != http.StatusCreated {
		t.Fatalf("lane 1 status = %d, body = %s", mint.Code, mint.Body.String())
	}
	var created createKeyResponse
	if err := json.Unmarshal(mint.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.Secret, "ak_") {
		t.Fatalf("secret = %q", created.Secret)
	}
	if created.AppID == nil || *created.AppID != defaultApp {
		t.Fatalf("stored app_id = %v", created.AppID)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM api_keys WHERE id = $1`, created.ID)
	})

	adminMintBody := `{"name":"bound-admin","operator_role_id":"` + operator.RoleIDAdmin.String() + `","app_id":"` + defaultApp.String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	adminMint := bindLiveDo(engine, http.MethodPost, "/admin/operator/keys", bindLiveEnvKey, adminMintBody)
	if adminMint.Code != http.StatusCreated {
		t.Fatalf("bound admin mint status = %d, body = %s", adminMint.Code, adminMint.Body.String())
	}
	var boundAdmin createKeyResponse
	if err := json.Unmarshal(adminMint.Body.Bytes(), &boundAdmin); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM api_keys WHERE id = $1`, boundAdmin.ID)
	})

	logs := bindLiveDo(engine, http.MethodGet, "/admin/activity-logs", created.Secret, "")
	saveBindLive(t, outDir, "bind-logs.json", logs)
	if logs.Code != http.StatusOK {
		t.Fatalf("lane 2 status = %d, body = %s", logs.Code, logs.Body.String())
	}
	if strings.Contains(logs.Body.String(), otherUser.String()) {
		t.Fatalf("logs leaked other app user: %s", logs.Body.String())
	}
	if !strings.Contains(logs.Body.String(), homeUser.String()) {
		t.Fatalf("logs missing bound app user: %s", logs.Body.String())
	}

	tenants := bindLiveDo(engine, http.MethodGet, "/admin/tenants", created.Secret, "")
	saveBindLive(t, outDir, "bind-tenants-403.json", tenants)
	if tenants.Code != http.StatusForbidden {
		t.Fatalf("lane 3 status = %d, body = %s", tenants.Code, tenants.Body.String())
	}

	ipRules := bindLiveDo(engine, http.MethodGet, "/admin/apps/"+otherApp.String()+"/ip-rules", boundAdmin.Secret, "")
	saveBindLive(t, outDir, "bind-ip-404.json", ipRules)
	if ipRules.Code != http.StatusNotFound {
		t.Fatalf("lane 4 status = %d, body = %s", ipRules.Code, ipRules.Body.String())
	}

	platformBody := `{"name":"platform-admin","operator_role_id":"` + operator.RoleIDSuperadmin.String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	platform := bindLiveDo(engine, http.MethodPost, "/admin/operator/keys", bindLiveEnvKey, platformBody)
	saveBindLive(t, outDir, "bind-platform.json", platform)
	if platform.Code != http.StatusCreated {
		t.Fatalf("lane 5 status = %d, body = %s", platform.Code, platform.Body.String())
	}
	var platformKey createKeyResponse
	if err := json.Unmarshal(platform.Body.Bytes(), &platformKey); err != nil {
		t.Fatal(err)
	}
	if platformKey.AppID != nil {
		t.Fatalf("platform app_id = %v", platformKey.AppID)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM api_keys WHERE id = $1`, platformKey.ID)
	})
	platformTenants := bindLiveDo(engine, http.MethodGet, "/admin/tenants", platformKey.Secret, "")
	if platformTenants.Code != http.StatusOK {
		t.Fatalf("lane 5 tenants status = %d, body = %s", platformTenants.Code, platformTenants.Body.String())
	}

	denied := bindLiveDo(engine, http.MethodPost, "/admin/operator/keys", bindLiveEnvKey,
		`{"name":"nope","operator_role_id":"`+operator.RoleIDSuperadmin.String()+`","app_id":"`+defaultApp.String()+`","expires_at":"`+operatorCreateKeyExpiry()+`"}`)
	saveBindLive(t, outDir, "bind-superadmin-400.json", denied)
	if denied.Code != http.StatusBadRequest {
		t.Fatalf("lane 6 status = %d, body = %s", denied.Code, denied.Body.String())
	}

	boundMint := bindLiveDo(engine, http.MethodPost, "/admin/operator/keys", boundAdmin.Secret,
		`{"name":"child","expires_at":"`+operatorCreateKeyExpiry()+`"}`)
	saveBindLive(t, outDir, "bind-cannot-mint.json", boundMint)
	if boundMint.Code != http.StatusForbidden {
		t.Fatalf("lane 7 status = %d, body = %s", boundMint.Code, boundMint.Body.String())
	}

	gui := gin.New()
	gui.DELETE("/gui/applications/:id", (&admin.GUIHandler{Repo: adminRepo}).AppDelete)
	delApp := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO applications (id, tenant_id, name) VALUES ($1, $2, 'delete-blocked')`, delApp, defaultApp); err != nil {
		t.Fatal(err)
	}
	raw, keyHash, prefix, suffix, err := admin.GenerateApiKey(admin.KeyTypeAdmin)
	if err != nil {
		t.Fatal(err)
	}
	_ = raw
	exp := time.Now().UTC().Add(90 * 24 * time.Hour)
	role := operator.RoleIDViewer
	if err := adminRepo.CreateApiKey(&models.ApiKey{
		KeyType:        admin.KeyTypeAdmin,
		Name:           "block-delete",
		KeyHash:        keyHash,
		KeyPrefix:      prefix,
		KeySuffix:      suffix,
		AppID:          &delApp,
		OperatorRoleID: &role,
		ExpiresAt:      &exp,
	}); err != nil {
		t.Fatal(err)
	}
	delRec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/gui/applications/"+delApp.String(), nil)
	gui.ServeHTTP(delRec, req)
	saveBindLive(t, outDir, "bind-app-delete-409.html", delRec)
	if delRec.Code != http.StatusConflict {
		t.Fatalf("lane 8 status = %d, body = %s", delRec.Code, delRec.Body.String())
	}
	if !strings.Contains(delRec.Body.String(), "bound admin keys") {
		t.Fatalf("lane 8 body = %s", delRec.Body.String())
	}
	var stillThere int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM applications WHERE id = $1`, delApp).Scan(&stillThere); err != nil {
		t.Fatal(err)
	}
	if stillThere != 1 {
		t.Fatal("application row was deleted")
	}

	replay := bindLiveDo(engine, http.MethodGet, "/admin/operator/keys/"+created.ID.String(), bindLiveEnvKey, "")
	saveBindLive(t, outDir, "bind-secret-once.json", replay)
	if replay.Code != http.StatusOK {
		t.Fatalf("lane 10 status = %d, body = %s", replay.Code, replay.Body.String())
	}
	if strings.Contains(replay.Body.String(), created.Secret) {
		t.Fatal("raw secret leaked on replay GET")
	}
}

func TestBindLive_PerfMint(t *testing.T) {
	h := iamEventTestEngine(t)
	appID := uuid.New()
	withApp := `{"name":"perf","app_id":"` + appID.String() + `","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	without := `{"name":"perf","expires_at":"` + operatorCreateKeyExpiry() + `"}`
	times := make([]time.Duration, 0, 10)
	controls := make([]time.Duration, 0, 10)
	for i := 0; i < 10; i++ {
		start := time.Now()
		rec := iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, withApp)
		elapsed := time.Since(start)
		if rec.Code != http.StatusCreated {
			t.Fatalf("with app_id status = %d", rec.Code)
		}
		times = append(times, elapsed)
		start = time.Now()
		rec = iamEventDo(h.engine, http.MethodPost, "/admin/operator/keys", accessSuperadminKey, without)
		elapsed = time.Since(start)
		if rec.Code != http.StatusCreated {
			t.Fatalf("without app_id status = %d", rec.Code)
		}
		controls = append(controls, elapsed)
	}
	head := medianDuration(times)
	base := medianDuration(controls)
	t.Logf("mint-with-app_id median=%s mint-without-app_id median=%s", head, base)
	if head > 2*base && base > 0 {
		t.Fatalf("head median %s exceeded twice baseline %s", head, base)
	}
}

func bindLiveDo(engine *gin.Engine, method, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Admin-API-Key", key)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func saveBindLive(t *testing.T, dir, name string, rec *httptest.ResponseRecorder) {
	t.Helper()
	payload := rec.Body.Bytes()
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("/tmp/swarm-bind", name), payload, 0o644); err != nil {
		t.Fatal(err)
	}
}

func bindLivePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	host := getenvDefault("DB_HOST", "localhost")
	port := getenvDefault("DB_PORT", "5432")
	user := getenvDefault("DB_USER", "postgres")
	pass := getenvDefault("DB_PASSWORD", "postgres")
	name := getenvDefault("DB_NAME", "auth_test")
	dsn := "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + name + "?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skip(err.Error())
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skip(err.Error())
	}
	return pool
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func medianDuration(in []time.Duration) time.Duration {
	if len(in) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), in...)
	for i := 0; i < len(cp); i++ {
		for j := i + 1; j < len(cp); j++ {
			if cp[j] < cp[i] {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	return cp[len(cp)/2]
}
