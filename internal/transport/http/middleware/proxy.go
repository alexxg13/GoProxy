package middleware

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"GoProxy/internal/models"
	"GoProxy/internal/service/accesslog"
	"GoProxy/internal/service/ipaccess"
	"GoProxy/internal/service/metrics"
	"GoProxy/pkg/cache"
	"GoProxy/pkg/ratelimit"

	"github.com/alexxg13/go-utils/logger"
	"github.com/gin-gonic/gin"
)

type responseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	bytes      int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

type ProxyMiddleware struct {
	ipService  ipaccess.IPService
	limiter    ratelimit.Limiter
	collector  metrics.Collector
	logStore   accesslog.Store
	zlog       logger.Logger
	cacheStore cache.Cache
}

func NewProxyMiddleware(
	ipService ipaccess.IPService,
	limiter ratelimit.Limiter,
	collector metrics.Collector,
	logStore accesslog.Store,
	zlog logger.Logger,
	cacheStore cache.Cache,
) *ProxyMiddleware {
	return &ProxyMiddleware{
		ipService:  ipService,
		limiter:    limiter,
		collector:  collector,
		logStore:   logStore,
		zlog:       zlog,
		cacheStore: cacheStore,
	}
}

func (m *ProxyMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/admin") ||
			strings.HasPrefix(c.Request.URL.Path, "/swagger") ||
			strings.HasPrefix(c.Request.URL.Path, "/captcha") ||
			c.Request.URL.Path == "/_info" ||
			c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		reqCtx := RequestCtx(c)
		AttachGinToRequest(c)
		ip := clientIP(c.Request)
		m.collector.IncConnections()
		defer m.collector.DecConnections()

		decision := m.ipService.Check(reqCtx, ip)
		if !decision.Allowed {
			m.collector.RecordSecurityEvent(string(decision.Reason))
			writeAccessLog(reqCtx, m.zlog, m.logStore, models.LogEntry{
				Level:   models.LogWarning,
				Type:    models.LogSecurity,
				IP:      ip,
				Message: string(decision.Reason),
				Method:  c.Request.Method,
				Path:    c.Request.URL.Path,
				Status:  http.StatusForbidden,
			})
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success":   false,
				"message":   string(decision.Reason),
				"client_ip": ip,
			})
			return
		}

		if !m.limiter.AllowConnection(ip) {
			m.collector.RecordSecurityEvent(string(models.DenyRateLimited))
			writeAccessLog(reqCtx, m.zlog, m.logStore, models.LogEntry{
				Level:   models.LogWarning,
				Type:    models.LogSecurity,
				IP:      ip,
				Message: string(models.DenyRateLimited),
				Method:  c.Request.Method,
				Path:    c.Request.URL.Path,
				Status:  http.StatusTooManyRequests,
			})
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		defer m.limiter.ReleaseConnection(ip)

		if !m.limiter.Allow(ip) {
			m.collector.RecordSecurityEvent(string(models.DenyRateLimited))
			writeAccessLog(reqCtx, m.zlog, m.logStore, models.LogEntry{
				Level:   models.LogWarning,
				Type:    models.LogSecurity,
				IP:      ip,
				Message: string(models.DenyRateLimited),
				Method:  c.Request.Method,
				Path:    c.Request.URL.Path,
				Status:  http.StatusTooManyRequests,
			})
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		cacheKey := m.cacheStore.Key(c.Request.Method, c.Request.Host, c.Request.URL.RequestURI())
		if c.Request.Method == http.MethodGet {
			if entry, ok := m.cacheStore.Get(cacheKey); ok {
				m.collector.RecordCacheEvent("hit")
				m.collector.RecordTraffic(len(entry.Body))
				m.collector.RecordClient(ip, len(entry.Body))
				for k, vals := range entry.Headers {
					for _, v := range vals {
						c.Writer.Header().Add(k, v)
					}
				}
				c.Data(entry.StatusCode, entry.Headers.Get("Content-Type"), entry.Body)
				m.logAccess(reqCtx, ip, c, entry.StatusCode, true)
				return
			}
			m.collector.RecordCacheEvent("miss")
		}

		start := time.Now()
		buf := &bytes.Buffer{}
		rw := &responseWriter{ResponseWriter: c.Writer, body: buf, statusCode: http.StatusOK}
		c.Writer = rw

		c.Next()

		latency := time.Since(start).Seconds() * 1000
		isError := rw.statusCode >= 400
		m.collector.RecordRequest(latency, isError)
		m.collector.RecordTraffic(rw.bytes)
		m.collector.RecordClient(ip, rw.bytes)

		if c.Request.Method == http.MethodGet && cache.ShouldCache(rw.statusCode, c.Writer.Header().Get("Cache-Control")) {
			body := buf.Bytes()
			if len(body) > 0 {
				headers := make(http.Header)
				for k, v := range c.Writer.Header() {
					headers[k] = v
				}
				m.cacheStore.Set(cacheKey, &cache.Entry{
					StatusCode: rw.statusCode,
					Headers:    headers,
					Body:       body,
				})
				m.collector.RecordCacheEvent("store")
			}
		}
		if c.Request.Method != http.MethodGet {
			cacheKeyGet := m.cacheStore.Key(http.MethodGet, c.Request.Host, c.Request.URL.RequestURI())
			if m.cacheStore.InvalidateKey(cacheKeyGet) {
				m.collector.RecordCacheEvent("invalidate")
			}
		}

		m.logAccess(reqCtx, ip, c, rw.statusCode, false)
	}
}

func (m *ProxyMiddleware) logAccess(reqCtx context.Context, ip string, c *gin.Context, status int, cached bool) {
	upstreamErr, _ := c.Get(lastUpstreamErrKey)
	errStr, _ := upstreamErr.(string)

	msg := accessLogMessage(status, cached, errStr)
	level := models.LogInfo
	logType := models.LogAccess
	if status >= 400 {
		level = models.LogWarning
	}
	if status >= 500 {
		level = models.LogWarning
		logType = models.LogErrorType
	}

	writeAccessLog(reqCtx, m.zlog, m.logStore, models.LogEntry{
		Level:   level,
		Type:    logType,
		IP:      ip,
		Message: msg,
		Method:  c.Request.Method,
		Path:    c.Request.URL.Path,
		Status:  status,
	}, upstreamErrField(c))
}

func clientIP(r *http.Request) string {
	var raw string
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		raw = strings.TrimSpace(parts[0])
	} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
		raw = strings.TrimSpace(xri)
	} else {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			raw = r.RemoteAddr
		} else {
			raw = host
		}
	}
	return strings.TrimPrefix(strings.TrimSuffix(raw, "]"), "[")
}
