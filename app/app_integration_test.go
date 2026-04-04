//go:build integration

package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	core "github.com/JedidiahDigital/go-core"
)

// =============================================================================
// Public API Integration Tests
// =============================================================================
//
// These tests exercise the full consumer-facing lifecycle:
//   app.New(cfg) → RegisterRoutes(r) → HTTP requests → Close()
//
// Requires PostgreSQL and Redis (or nil Redis for in-memory mode).
//
// Run with:
//   go test -v -tags=integration ./app/...
//
// Environment variables (match CI services):
//   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
//   REDIS_ADDR (optional — omit for in-memory cache)
//
// =============================================================================

// testConfig builds a Config from environment variables, falling back to
// sensible defaults that match the CI workflow.
func testConfig() core.Config {
	cfg := core.DefaultConfig()
	cfg.Database.Host = envOr("DB_HOST", "localhost")
	cfg.Database.Port = envOrInt("DB_PORT", 5432)
	cfg.Database.User = envOr("DB_USER", "postgres")
	cfg.Database.Password = envOr("DB_PASSWORD", "postgres")
	cfg.Database.DBName = envOr("DB_NAME", "auth_test")
	cfg.Database.SSLMode = "disable"
	cfg.JWT.Secret = "integration-test-secret-that-is-at-least-32-bytes-long!"
	cfg.JWT.AccessTokenTTL = 15 * time.Minute
	cfg.JWT.RefreshTokenTTL = 720 * time.Hour
	cfg.GinMode = "test"

	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		cfg.Redis = &core.RedisConfig{Addr: addr}
	} else {
		cfg.Redis = nil // in-memory cache
	}

	cfg.Email = nil // no SMTP in tests
	return cfg
}

func TestNew_InvalidConfig(t *testing.T) {
	cfg := core.Config{} // missing everything
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for empty config, got nil")
	}
}

func TestNew_BadDatabase(t *testing.T) {
	cfg := testConfig()
	cfg.Database.Host = "192.0.2.1" // unreachable (TEST-NET)
	cfg.Database.Port = 1

	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for unreachable database, got nil")
	}
}

func TestLifecycle(t *testing.T) {
	cfg := testConfig()

	a, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer a.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	a.RegisterRoutes(r)

	// Verify health endpoint is registered and responds
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /health: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestLifecycle_PublicRoutesRegistered(t *testing.T) {
	cfg := testConfig()

	a, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer a.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	a.RegisterRoutes(r)

	// These public endpoints should exist and return something other than 404.
	// We don't care about the response body — just that the route is registered.
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodPost, "/register"},
		{http.MethodPost, "/login"},
		{http.MethodPost, "/refresh-token"},
		{http.MethodPost, "/forgot-password"},
		{http.MethodPost, "/magic-link/request"},
		{http.MethodGet, "/2fa/methods"},
	}

	for _, rt := range routes {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(rt.method, rt.path, nil)
		r.ServeHTTP(w, req)

		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s returned 404 — route not registered", rt.method, rt.path)
		}
	}
}

func TestClose_Idempotent(t *testing.T) {
	cfg := testConfig()

	a, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Close twice should not panic
	a.Close()
	a.Close()
}

// --- helpers -----------------------------------------------------------------

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}
