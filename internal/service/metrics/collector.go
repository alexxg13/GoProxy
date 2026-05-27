package metrics

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"GoProxy/internal/models"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Collector interface {
	RecordRequest(latencyMs float64, isError bool)
	RecordTraffic(bytes int)
	RecordClient(ip string, bytes int)
	RecordCacheEvent(event string)
	RecordSecurityEvent(reason string)
	IncConnections()
	DecConnections()
	ProxyMetrics() models.ProxyMetrics
	SystemMetrics() models.SystemMetrics
}
type CollectorImpl struct {
	totalRequests     atomic.Int64
	errorCount        atomic.Int64
	activeConnections atomic.Int64
	latencies         []float64
	latMu             sync.Mutex
	startTime         time.Time
}

var (
	proxyRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goproxy_proxy_requests_total",
		Help: "Total proxied HTTP requests by result.",
	}, []string{"result"})
	proxyRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "goproxy_proxy_request_duration_seconds",
		Help:    "Proxied request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	})
	proxyActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "goproxy_proxy_active_connections",
		Help: "Current active proxy connections.",
	})
	proxyTrafficBytesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "goproxy_proxy_traffic_bytes_total",
		Help: "Total response bytes sent by the proxy.",
	})
	clientRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goproxy_client_requests_total",
		Help: "Total requests by client IP.",
	}, []string{"ip"})
	clientTrafficBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goproxy_client_traffic_bytes_total",
		Help: "Total response bytes by client IP.",
	}, []string{"ip"})
	cacheEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goproxy_cache_events_total",
		Help: "HTTP cache events.",
	}, []string{"event"})
	securityEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goproxy_security_events_total",
		Help: "Security and filtering events.",
	}, []string{"reason"})
)

func NewCollector() *CollectorImpl {
	return &CollectorImpl{
		latencies: make([]float64, 0, 10000),
		startTime: time.Now(),
	}
}

func (c *CollectorImpl) RecordRequest(latencyMs float64, isError bool) {
	c.totalRequests.Add(1)
	result := "success"
	if isError {
		c.errorCount.Add(1)
		result = "error"
	}
	proxyRequestsTotal.WithLabelValues(result).Inc()
	proxyRequestDuration.Observe(latencyMs / 1000)
	c.latMu.Lock()
	if len(c.latencies) < 10000 {
		c.latencies = append(c.latencies, latencyMs)
	}
	c.latMu.Unlock()
}

func (c *CollectorImpl) RecordTraffic(bytes int) {
	if bytes > 0 {
		proxyTrafficBytesTotal.Add(float64(bytes))
	}
}

func (c *CollectorImpl) RecordClient(ip string, bytes int) {
	clientRequestsTotal.WithLabelValues(ip).Inc()
	if bytes > 0 {
		clientTrafficBytesTotal.WithLabelValues(ip).Add(float64(bytes))
	}
}

func (c *CollectorImpl) RecordCacheEvent(event string) {
	cacheEventsTotal.WithLabelValues(event).Inc()
}

func (c *CollectorImpl) RecordSecurityEvent(reason string) {
	securityEventsTotal.WithLabelValues(reason).Inc()
}

func (c *CollectorImpl) IncConnections() {
	c.activeConnections.Add(1)
	proxyActiveConnections.Inc()
}

func (c *CollectorImpl) DecConnections() {
	c.activeConnections.Add(-1)
	proxyActiveConnections.Dec()
}

func (c *CollectorImpl) ProxyMetrics() models.ProxyMetrics {
	total := c.totalRequests.Load()
	errors := c.errorCount.Load()

	elapsed := time.Since(c.startTime).Seconds()
	rps := float64(0)
	if elapsed > 0 {
		rps = float64(total) / elapsed
	}

	c.latMu.Lock()
	lats := append([]float64(nil), c.latencies...)
	c.latMu.Unlock()

	avg, p99 := calcLatency(lats)
	errRate := float64(0)
	if total > 0 {
		errRate = float64(errors) / float64(total) * 100
	}

	return models.ProxyMetrics{
		TotalRequests:     total,
		RequestsPerSecond: rps,
		AvgLatency:        avg,
		P99Latency:        p99,
		ErrorRate:         errRate,
		ActiveConnections: c.activeConnections.Load(),
	}
}

func (c *CollectorImpl) SystemMetrics() models.SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memPercent := float64(m.Alloc) / float64(m.Sys) * 100
	if m.Sys == 0 {
		memPercent = 0
	}

	return models.SystemMetrics{
		CPU:        0,
		Memory:     memPercent,
		Goroutines: runtime.NumGoroutine(),
		GCPauses:   float64(m.PauseTotalNs) / 1e6,
	}
}

func calcLatency(lats []float64) (avg, p99 float64) {
	if len(lats) == 0 {
		return 0, 0
	}
	var sum float64
	for _, l := range lats {
		sum += l
	}
	avg = sum / float64(len(lats))

	sorted := append([]float64(nil), lats...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	idx := int(float64(len(sorted)) * 0.99)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	p99 = sorted[idx]
	return avg, p99
}
