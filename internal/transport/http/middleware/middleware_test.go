package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"GoProxy/internal/models"
	_ "GoProxy/internal/service/accesslog"
	"GoProxy/internal/service/ipaccess"
	_ "GoProxy/internal/service/metrics"
	"GoProxy/pkg/cache"
	"GoProxy/pkg/ratelimit"
	"github.com/alexxg13/go-utils/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type mockIPService struct {
	checkFunc func(ctx context.Context, ip string) ipaccess.Decision
}

func (m *mockIPService) Check(ctx context.Context, ip string) ipaccess.Decision {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, ip)
	}
	return ipaccess.Decision{Allowed: true}
}
func (m *mockIPService) VerifyCaptcha(ctx context.Context, ip string, answer string) ipaccess.Decision {
	return ipaccess.Decision{Allowed: true}
}
func (m *mockIPService) Create(ctx context.Context, rule models.IPRuleCreate) (models.IPRule, error) {
	return models.IPRule{}, nil
}
func (m *mockIPService) Update(ctx context.Context, id string, rule models.IPRuleCreate) error {
	return nil
}
func (m *mockIPService) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockIPService) Get(ctx context.Context, id string) (models.IPRule, error) {
	return models.IPRule{}, nil
}
func (m *mockIPService) List(ruleType models.RuleType) []models.IPRule {
	return nil
}

type mockLimiter struct {
	allowFunc           func(ip string) bool
	allowConnectionFunc func(ip string) bool
	releasedIP          string
}

func (m *mockLimiter) Allow(ip string) bool {
	if m.allowFunc != nil {
		return m.allowFunc(ip)
	}
	return true
}
func (m *mockLimiter) AllowConnection(ip string) bool {
	if m.allowConnectionFunc != nil {
		return m.allowConnectionFunc(ip)
	}
	return true
}
func (m *mockLimiter) ReleaseConnection(ip string) {
	m.releasedIP = ip
}
func (m *mockLimiter) Config() ratelimit.Config {
	return ratelimit.Config{}
}
func (m *mockLimiter) UpdateConfig(cfg ratelimit.Config) {}
func (m *mockLimiter) Violators() []ratelimit.Violation {
	return nil
}

type mockCollector struct {
	incConnsCalled      bool
	decConnsCalled      bool
	recordRequestCalled bool
}

func (m *mockCollector) IncConnections() { m.incConnsCalled = true }
func (m *mockCollector) DecConnections() { m.decConnsCalled = true }
func (m *mockCollector) RecordRequest(latencyMs float64, isError bool) {
	m.recordRequestCalled = true
}
func (m *mockCollector) RecordTraffic(bytes int)           {}
func (m *mockCollector) RecordClient(ip string, bytes int) {}
func (m *mockCollector) RecordCacheEvent(event string)     {}
func (m *mockCollector) RecordSecurityEvent(reason string) {}
func (m *mockCollector) ProxyMetrics() models.ProxyMetrics {
	return models.ProxyMetrics{}
}
func (m *mockCollector) CacheMetrics() models.CacheMetrics {
	return models.CacheMetrics{}
}
func (m *mockCollector) SystemMetrics() models.SystemMetrics {
	return models.SystemMetrics{}
}

type mockLogStore struct {
	entries []models.LogEntry
}

func (m *mockLogStore) Add(entry models.LogEntry) {
	m.entries = append(m.entries, entry)
}
func (m *mockLogStore) Close() error { return nil }
func (m *mockLogStore) List(level models.LogLevel, logType models.LogType, limit int) []models.LogEntry {
	return m.entries
}

type mockLogger struct{}

func (m *mockLogger) Sync() error {
	panic("implement me")
}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...zap.Field) {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...zap.Field)  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...zap.Field)  {}
func (m *mockLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {}
func (m *mockLogger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {}
func (m *mockLogger) With(fields ...zap.Field) logger.Logger                     { return m }

type mockCache struct {
	store          map[string]*cache.Entry
	invalidatedKey string
	invalidatedCnt int
	defaultTTL     time.Duration
}

func (m *mockCache) Key(method, host, path string) string {
	return method + "|" + host + "|" + path
}
func (m *mockCache) Get(key string) (*cache.Entry, bool) {
	entry, ok := m.store[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry, true
}
func (m *mockCache) Set(key string, entry *cache.Entry) {
	if m.store == nil {
		m.store = make(map[string]*cache.Entry)
	}
	if entry.ExpiresAt.IsZero() {
		ttl := m.defaultTTL
		if ttl == 0 {
			ttl = 5 * time.Minute
		}
		entry.ExpiresAt = time.Now().Add(ttl)
	}
	m.store[key] = entry
}
func (m *mockCache) InvalidateKey(key string) bool {
	m.invalidatedKey = key
	m.invalidatedCnt++
	delete(m.store, key)
	return true
}
func (m *mockCache) InvalidatePrefix(prefix string) int { return 0 }
func (m *mockCache) InvalidateRegex(pattern string) (int, error) {
	return 0, nil
}
func (m *mockCache) InvalidateTags(tags []string) int { return 0 }
func (m *mockCache) Clear()                           {}
func (m *mockCache) Metrics() (int64, int64, int64, string) {
	return 0, 0, 0, "0B"
}

func setupTest() (*gin.Engine, *ProxyMiddleware, *mockCollector, *mockCache, *mockLogStore, *mockIPService, *mockLimiter) {
	gin.SetMode(gin.TestMode)

	mockIP := &mockIPService{}
	mockLimit := &mockLimiter{}
	mockColl := &mockCollector{}
	mockLog := &mockLogStore{}
	mockLogg := &mockLogger{}
	mockCch := &mockCache{
		store:      make(map[string]*cache.Entry),
		defaultTTL: 5 * time.Minute,
	}

	mw := NewProxyMiddleware(mockIP, mockLimit, mockColl, mockLog, mockLogg, mockCch)

	router := gin.New()
	router.Use(mw.Handle())
	router.Any("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.Any("/admin/foo", func(c *gin.Context) {
		c.String(http.StatusOK, "admin")
	})

	return router, mw, mockColl, mockCch, mockLog, mockIP, mockLimit
}

func TestMiddleware_SkipSystemPaths(t *testing.T) {
	router, _, _, _, _, _, _ := setupTest()

	req := httptest.NewRequest(http.MethodGet, "/admin/foo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "admin" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestMiddleware_IPBlocked(t *testing.T) {
	router, _, _, _, mockLog, mockIP, _ := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: false, Reason: models.DenyBlacklisted}
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if len(mockLog.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(mockLog.entries))
	}
	if mockLog.entries[0].Status != http.StatusForbidden {
		t.Errorf("log status mismatch")
	}
}

func TestMiddleware_RateLimitExceeded(t *testing.T) {
	router, _, _, _, mockLog, mockIP, mockLimit := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: true}
	}
	mockLimit.allowFunc = func(ip string) bool {
		return false
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if len(mockLog.entries) != 1 {
		t.Fatal("log entry missing")
	}
}

func TestMiddleware_MaxConnectionsExceeded(t *testing.T) {
	router, _, _, _, mockLog, mockIP, mockLimit := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: true}
	}
	mockLimit.allowConnectionFunc = func(ip string) bool {
		return false
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if len(mockLog.entries) != 1 {
		t.Fatal("log entry missing")
	}
	if mockLimit.releasedIP != "" {
		t.Fatal("connection should not be released when it was not acquired")
	}
}

func TestMiddleware_ReleasesConnectionAfterRequest(t *testing.T) {
	router, _, _, _, _, mockIP, mockLimit := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: true}
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.2")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mockLimit.releasedIP != "10.0.0.2" {
		t.Fatalf("connection was not released, got %q", mockLimit.releasedIP)
	}
}

func TestMiddleware_CacheHitForGet(t *testing.T) {
	_, mw, _, mockCache, mockLog, _, _ := setupTest()
	cacheKey := mw.cacheStore.Key(http.MethodGet, "example.com", "/test")
	entry := &cache.Entry{
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("cached body"),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	mockCache.Set(cacheKey, entry)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Host = "example.com"
	c.Request.Header.Set("X-Real-IP", "1.1.1.1")
	mw.Handle()(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "cached body" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
	if len(mockLog.entries) != 1 {
		t.Fatal("log not written")
	}
	if mockLog.entries[0].Message != "cache hit" {
		t.Errorf("log message not from cached: %s", mockLog.entries[0].Message)
	}
}

func TestMiddleware_CacheMissThenStore(t *testing.T) {
	router, mw, mockColl, mockCache, _, mockIP, mockLimit := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: true}
	}
	mockLimit.allowFunc = func(ip string) bool { return true }

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "example.com"
	req.Header.Set("X-Real-IP", "2.2.2.2")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	key := mw.cacheStore.Key(http.MethodGet, "example.com", "/test")
	entry, ok := mockCache.Get(key)
	if !ok {
		t.Fatalf("cache entry not stored; store keys: %v", mockCache.store)
	}
	if string(entry.Body) != "OK" {
		t.Errorf("cache body mismatch: %s", string(entry.Body))
	}
	if !mockColl.recordRequestCalled {
		t.Error("RecordRequest not called")
	}
}

func TestMiddleware_CacheKeyIncludesQuery(t *testing.T) {
	router, mw, _, mockCache, _, mockIP, mockLimit := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: true}
	}
	mockLimit.allowFunc = func(ip string) bool { return true }

	req := httptest.NewRequest(http.MethodGet, "/test?page=2", nil)
	req.Host = "example.com"
	req.Header.Set("X-Real-IP", "2.2.2.2")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	key := mw.cacheStore.Key(http.MethodGet, "example.com", "/test?page=2")
	if _, ok := mockCache.Get(key); !ok {
		t.Fatalf("cache entry with query not stored; store keys: %v", mockCache.store)
	}
	withoutQuery := mw.cacheStore.Key(http.MethodGet, "example.com", "/test")
	if _, ok := mockCache.Get(withoutQuery); ok {
		t.Fatal("cache entry should not be stored under path without query")
	}
}

func TestMiddleware_InvalidateCacheOnNonGet(t *testing.T) {
	router, mw, _, mockCache, _, mockIP, mockLimit := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: true}
	}
	mockLimit.allowFunc = func(ip string) bool { return true }

	getKey := mw.cacheStore.Key(http.MethodGet, "example.com", "/test")
	mockCache.Set(getKey, &cache.Entry{StatusCode: 200, Body: []byte("old"), ExpiresAt: time.Now().Add(time.Minute)})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Host = "example.com"
	req.Header.Set("X-Real-IP", "3.3.3.3")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	_, ok := mockCache.Get(getKey)
	if ok {
		t.Error("cache key was not invalidated after non-GET")
	}
	if mockCache.invalidatedKey != getKey {
		t.Errorf("invalidateKey called with wrong key: got %s, want %s", mockCache.invalidatedKey, getKey)
	}
}

func TestMiddleware_InvalidateCacheOnNonGetIncludesQuery(t *testing.T) {
	router, mw, _, mockCache, _, mockIP, mockLimit := setupTest()
	mockIP.checkFunc = func(ctx context.Context, ip string) ipaccess.Decision {
		return ipaccess.Decision{Allowed: true}
	}
	mockLimit.allowFunc = func(ip string) bool { return true }

	getKey := mw.cacheStore.Key(http.MethodGet, "example.com", "/test?id=42")
	mockCache.Set(getKey, &cache.Entry{StatusCode: 200, Body: []byte("old"), ExpiresAt: time.Now().Add(time.Minute)})

	req := httptest.NewRequest(http.MethodPost, "/test?id=42", nil)
	req.Host = "example.com"
	req.Header.Set("X-Real-IP", "3.3.3.3")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if mockCache.invalidatedKey != getKey {
		t.Errorf("invalidateKey called with wrong key: got %s, want %s", mockCache.invalidatedKey, getKey)
	}
}
