package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"GoProxy/internal/service/metrics"
	"GoProxy/pkg/cache"

	"github.com/gin-gonic/gin"
)

func TestMetricsHandlerInvalidateCacheByKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := cache.NewStore(cache.Config{})
	store.Set("cache-key", &cache.Entry{Body: []byte("ok")})
	h := NewMetricsHandler(metrics.NewCollector(), store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/cache/invalidate", bytes.NewBufferString(`{"mode":"key","key":"cache-key"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.InvalidateCache(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if _, ok := store.Get("cache-key"); ok {
		t.Fatal("cache key should be invalidated")
	}
}

func TestMetricsHandlerInvalidateCacheBadRegex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewMetricsHandler(metrics.NewCollector(), cache.NewStore(cache.Config{}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/cache/invalidate", bytes.NewBufferString(`{"mode":"regex","pattern":"["}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.InvalidateCache(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}
