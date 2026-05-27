package v1

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

type testMetricsHandler struct{}

func (testMetricsHandler) GetProxyMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "proxy"})
}
func (testMetricsHandler) GetCacheMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "cache"})
}
func (testMetricsHandler) InvalidateCache(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "invalidate"})
}
func (testMetricsHandler) GetSystemMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "system"})
}

type testIPHandler struct{}

func (testIPHandler) GetIPRules(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"handler": "ip-rules"}) }
func (testIPHandler) CheckIPAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "ip-check"})
}
func (testIPHandler) VerifyCaptcha(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "captcha"})
}
func (testIPHandler) CreateIPRule(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"handler": "ip-create"})
}
func (testIPHandler) DeleteIPRule(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "ip-delete"})
}

type testLogsHandler struct{}

func (testLogsHandler) GetLogs(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"handler": "logs"}) }

type testUpstreamHandler struct{}

func (testUpstreamHandler) GetUpstreamServers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "upstream"})
}

type testRateHandler struct{}

func (testRateHandler) GetRateLimitConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "rate-config"})
}
func (testRateHandler) UpdateRateLimitConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "rate-update"})
}
func (testRateHandler) GetRateViolators(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"handler": "rate-violators"})
}

func TestNewHandlers(t *testing.T) {
	handlers := NewHandlers(testMetricsHandler{}, testIPHandler{}, testLogsHandler{}, testUpstreamHandler{}, testRateHandler{})
	if handlers.MetricsHandler == nil || handlers.IPHandler == nil || handlers.RatelimiterHandler == nil {
		t.Fatal("expected handlers to be assigned")
	}
}

func TestRegisterHandlersRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	target, _ := url.Parse("http://backend.test")
	server := &Server{
		engine: gin.New(),
		proxy:  httputil.NewSingleHostReverseProxy(target),
		handlers: NewHandlers(
			testMetricsHandler{},
			testIPHandler{},
			testLogsHandler{},
			testUpstreamHandler{},
			testRateHandler{},
		),
	}
	server.RegisterHandlers()

	for _, tc := range []struct {
		method string
		path   string
		code   int
	}{
		{http.MethodGet, "/_info", http.StatusOK},
		{http.MethodPost, "/captcha/verify", http.StatusOK},
		{http.MethodGet, "/admin/metrics/proxy", http.StatusOK},
		{http.MethodGet, "/admin/metrics/cache", http.StatusOK},
		{http.MethodPost, "/admin/cache/invalidate", http.StatusOK},
		{http.MethodGet, "/admin/ip-rules", http.StatusOK},
		{http.MethodGet, "/admin/ip-rules/check?ip=1.2.3.4", http.StatusOK},
		{http.MethodPost, "/admin/ip-rules", http.StatusCreated},
		{http.MethodDelete, "/admin/ip-rules/1", http.StatusOK},
		{http.MethodGet, "/admin/logs", http.StatusOK},
		{http.MethodGet, "/admin/upstream-servers", http.StatusOK},
		{http.MethodGet, "/admin/rate-limit/config", http.StatusOK},
		{http.MethodPatch, "/admin/rate-limit/config", http.StatusOK},
		{http.MethodGet, "/admin/rate-limit/violators", http.StatusOK},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		server.engine.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("%s %s: expected %d, got %d body=%s", tc.method, tc.path, tc.code, w.Code, w.Body.String())
		}
	}
}

func TestServerShutdownWithoutRun(t *testing.T) {
	server := &Server{srv: &http.Server{}}
	if err := server.Shutdown(httptest.NewRequest(http.MethodGet, "/", nil).Context()); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}
