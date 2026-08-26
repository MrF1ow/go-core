package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/geoip"
)

func doIPRuleRequest(evaluate func(uuid.UUID, string) geoip.AccessResult, appID *uuid.UUID, clientIP string) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		if appID != nil {
			c.Set(AppIDKey, *appID)
		}
		IPRuleMiddleware(evaluate)(c)
	}, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if clientIP != "" {
		req.Header.Set("X-Forwarded-For", clientIP)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestIPRule_NilEvaluator(t *testing.T) {
	appID := uuid.New()
	w := doIPRuleRequest(nil, &appID, "1.2.3.4")
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for nil evaluator, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIPRule_Allowed(t *testing.T) {
	appID := uuid.New()
	evaluate := func(uuid.UUID, string) geoip.AccessResult {
		return geoip.AccessResult{Allowed: true}
	}
	w := doIPRuleRequest(evaluate, &appID, "1.2.3.4")
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 when allowed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIPRule_Denied(t *testing.T) {
	appID := uuid.New()
	evaluate := func(uuid.UUID, string) geoip.AccessResult {
		return geoip.AccessResult{Allowed: false}
	}
	w := doIPRuleRequest(evaluate, &appID, "1.2.3.4")
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 when denied, got %d: %s", w.Code, w.Body.String())
	}
	if errMsg := jsonBody(w); errMsg != "Access denied from your location" {
		t.Fatalf("Expected 'Access denied from your location', got: %s", errMsg)
	}
}

func TestIPRule_AllowCIDRHit(t *testing.T) {
	appID := uuid.New()
	evaluate := func(_ uuid.UUID, clientIP string) geoip.AccessResult {
		return geoip.AccessResult{Allowed: clientIP == "10.0.0.5"}
	}
	w := doIPRuleRequest(evaluate, &appID, "10.0.0.5")
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for allow CIDR hit, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIPRule_AllowCIDRMiss(t *testing.T) {
	appID := uuid.New()
	evaluate := func(_ uuid.UUID, clientIP string) geoip.AccessResult {
		return geoip.AccessResult{Allowed: clientIP == "10.0.0.5"}
	}
	w := doIPRuleRequest(evaluate, &appID, "8.8.8.8")
	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 for allow CIDR miss, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIPRule_MissingAppID(t *testing.T) {
	evaluate := func(uuid.UUID, string) geoip.AccessResult {
		t.Fatal("evaluate must not run without app_id")
		return geoip.AccessResult{Allowed: true}
	}
	w := doIPRuleRequest(evaluate, nil, "1.2.3.4")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for missing context app_id, got %d: %s", w.Code, w.Body.String())
	}
}
