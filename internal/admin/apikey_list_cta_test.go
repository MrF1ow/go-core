package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/web"
)

func TestApiKeyListTemplate_WriteHasRevokeNotDelete(t *testing.T) {
	item := ApiKeyListItem{
		ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		KeyType:   KeyTypeAdmin,
		Name:      "ops",
		KeyPrefix: "ak_test",
		KeySuffix: "abcd",
		CreatedAt: time.Now(),
	}
	withWrite := renderApiKeyList(t, gin.H{
		"Keys":       []ApiKeyListItem{item},
		"Page":       1,
		"TotalPages": 1,
		"Total":      1,
		"CanWrite":   true,
	})
	if !strings.Contains(withWrite, "bi-shield-x") {
		t.Fatalf("writer missing revoke: %s", withWrite)
	}
	if strings.Contains(withWrite, "bi-trash") || strings.Contains(withWrite, "/delete") {
		t.Fatalf("writer still has delete: %s", withWrite)
	}
	withoutWrite := renderApiKeyList(t, gin.H{
		"Keys":       []ApiKeyListItem{item},
		"Page":       1,
		"TotalPages": 1,
		"Total":      1,
		"CanWrite":   false,
	})
	if strings.Contains(withoutWrite, "bi-shield-x") || strings.Contains(withoutWrite, "bi-trash") {
		t.Fatalf("reader saw write actions: %s", withoutWrite)
	}
}

func renderApiKeyList(t *testing.T, data gin.H) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	renderer, err := web.NewRenderer("/gui")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.HTMLRender = renderer
	router.GET("/list", func(c *gin.Context) {
		c.HTML(http.StatusOK, "api_key_list", data)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/list", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "ops") {
		t.Fatalf("truncated or empty body = %s", body)
	}
	return body
}
