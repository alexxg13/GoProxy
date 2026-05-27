package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"GoProxy/internal/models"
	"GoProxy/internal/service/accesslog"
	"GoProxy/internal/service/ipaccess"
	"GoProxy/internal/service/metrics"
	"GoProxy/pkg/cache"
	"GoProxy/pkg/ratelimit"

	"github.com/gin-gonic/gin"
)

type testChecker struct {
	servers []models.UpstreamServer
}

func (t testChecker) List() []models.UpstreamServer {
	return t.servers
}

func testContext(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return c, w
}

func TestLogsHandlerGetLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := accesslog.NewStore(10)
	store.Add(models.LogEntry{Level: models.LogInfo, Type: models.LogAccess, IP: "1.1.1.1", Message: "ok"})
	h := NewLogsHandler(store)

	c, w := testContext(http.MethodGet, "/admin/logs?level=info&type=access&limit=5", "")
	h.GetLogs(c)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "1.1.1.1") {
		t.Fatalf("unexpected response: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestMetricsHandlerEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	collector := metrics.NewCollector()
	collector.RecordRequest(12, false)
	collector.IncConnections()
	collector.DecConnections()
	store := cache.NewStore(cache.Config{})
	store.Set("k", &cache.Entry{Body: []byte("body")})
	h := NewMetricsHandler(collector, store)

	for _, tc := range []struct {
		name string
		call func(*gin.Context)
		path string
	}{
		{name: "proxy", call: h.GetProxyMetrics, path: "/admin/metrics/proxy"},
		{name: "cache", call: h.GetCacheMetrics, path: "/admin/metrics/cache"},
		{name: "system", call: h.GetSystemMetrics, path: "/admin/metrics/system"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, w := testContext(http.MethodGet, tc.path, "")
			tc.call(c)
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestMetricsHandlerInvalidateModes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := cache.NewStore(cache.Config{})
	store.Set("prefix:1", &cache.Entry{Body: []byte("1"), Tags: []string{"tag"}})
	store.Set("prefix:2", &cache.Entry{Body: []byte("2")})
	store.Set("other:1", &cache.Entry{Body: []byte("3")})
	h := NewMetricsHandler(metrics.NewCollector(), store)

	for _, body := range []string{
		`{"mode":"prefix","prefix":"prefix:"}`,
		`{"mode":"regex","pattern":"^other:"}`,
		`{"mode":"tags","tags":["tag"]}`,
		`{"mode":"clear"}`,
	} {
		store.Set("prefix:1", &cache.Entry{Body: []byte("1"), Tags: []string{"tag"}})
		store.Set("prefix:2", &cache.Entry{Body: []byte("2")})
		store.Set("other:1", &cache.Entry{Body: []byte("3")})

		c, w := testContext(http.MethodPost, "/admin/cache/invalidate", body)
		h.InvalidateCache(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", body, w.Code, w.Body.String())
		}
	}
}

func TestMetricsHandlerInvalidateValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewMetricsHandler(metrics.NewCollector(), cache.NewStore(cache.Config{}))

	for _, body := range []string{
		`{"mode":"key"}`,
		`{"mode":"prefix"}`,
		`{"mode":"regex"}`,
		`{"mode":"tags"}`,
		`{"mode":"unknown"}`,
		`{`,
	} {
		c, w := testContext(http.MethodPost, "/admin/cache/invalidate", body)
		h.InvalidateCache(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", body, w.Code)
		}
	}
}

func TestRatelimiterHandlerEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := ratelimit.New(ratelimit.Config{RPS: 1, RPM: 2, MaxConnections: 3, BanDuration: time.Second})
	limiter.Allow("1.2.3.4")
	limiter.Allow("1.2.3.4")
	h := NewRatelimiterHandler(limiter)

	c, w := testContext(http.MethodGet, "/admin/rate-limit/config", "")
	h.GetRateLimitConfig(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	c, w = testContext(http.MethodPatch, "/admin/rate-limit/config", `{"rps":10,"rpm":20,"maxConnections":30,"banDuration":40}`)
	h.UpdateRateLimitConfig(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	c, w = testContext(http.MethodPatch, "/admin/rate-limit/config", `{`)
	h.UpdateRateLimitConfig(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	c, w = testContext(http.MethodGet, "/admin/rate-limit/violators", "")
	h.GetRateViolators(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUpstreamHandlerGetUpstreamServers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUpstreamHandler(testChecker{servers: []models.UpstreamServer{{
		Name: "backend", URL: "http://backend", Status: models.UpstreamHealthy, LastCheck: time.Now(),
	}}})

	c, w := testContext(http.MethodGet, "/admin/upstream-servers", "")
	h.GetUpstreamServers(c)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "backend") {
		t.Fatalf("unexpected response: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestConstructors(t *testing.T) {
	if NewLogsHandler(accesslog.NewStore(1)) == nil {
		t.Fatal("logs handler is nil")
	}
	if NewRatelimiterHandler(ratelimit.New(ratelimit.Config{})) == nil {
		t.Fatal("ratelimit handler is nil")
	}
	if NewUpstreamHandler(testChecker{}) == nil {
		t.Fatal("upstream handler is nil")
	}
}

func TestIPHandlerDeleteRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccessServiceForDelete()
	h := NewIPHandler(svc)

	rule, err := svc.Create(context.Background(), models.IPRuleCreate{IP: "10.0.0.1", Type: models.RuleAllow, Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}

	c, w := testContext(http.MethodDelete, "/admin/ip-rules/"+rule.ID, "")
	c.Params = gin.Params{{Key: "ruleId", Value: rule.ID}}
	h.DeleteIPRule(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	c, w = testContext(http.MethodDelete, "/admin/ip-rules/missing", "")
	c.Params = gin.Params{{Key: "ruleId", Value: "missing"}}
	h.DeleteIPRule(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestIPHandlerCreateValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccessServiceForDelete()
	h := NewIPHandler(svc)

	c, w := testContext(http.MethodPost, "/admin/ip-rules", `{`)
	h.CreateIPRule(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	c, w = testContext(http.MethodPost, "/admin/ip-rules", `{"ip":"bad","type":"allow","reason":"test"}`)
	h.CreateIPRule(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func ipaccessServiceForDelete() (*ipaccess.IPServiceImpl, error) {
	return ipaccess.New(ipaccess.Config{DefaultDeny: true})
}
