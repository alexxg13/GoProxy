package dto

import (
	"time"

	"GoProxy/internal/models"
)

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type DataResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type IPRule struct {
	ID        string `json:"id"`
	IP        string `json:"ip"`
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	AddedDate string `json:"addedDate"`
	Requests  int64  `json:"requests"`
}

type IPRuleCreateRequest struct {
	IP     string `json:"ip" binding:"required"`
	Type   string `json:"type" binding:"required,oneof=allow deny grey"`
	Reason string `json:"reason" binding:"required"`
}

type IPAccessCheck struct {
	IP      string `json:"ip"`
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	Grey    bool   `json:"grey"`
}

type CaptchaVerifyRequest struct {
	Answer string `json:"answer" binding:"required"`
}

type CaptchaVerifyResult struct {
	IP      string `json:"ip"`
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type ProxyMetrics struct {
	TotalRequests     int64   `json:"totalRequests"`
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	AvgLatency        float64 `json:"avgLatency"`
	P99Latency        float64 `json:"p99Latency"`
	ErrorRate         float64 `json:"errorRate"`
	ActiveConnections int64   `json:"activeConnections"`
}

type CacheMetrics struct {
	Hits          int64  `json:"hits"`
	Misses        int64  `json:"misses"`
	Size          string `json:"size"`
	Invalidations int64  `json:"invalidations"`
}

type CacheInvalidationRequest struct {
	Mode    string   `json:"mode" binding:"required,oneof=key prefix regex tags clear"`
	Key     string   `json:"key,omitempty"`
	Prefix  string   `json:"prefix,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

type CacheInvalidationResult struct {
	Mode    string `json:"mode"`
	Deleted int    `json:"deleted"`
}

type SystemMetrics struct {
	CPU        float64 `json:"cpu"`
	Memory     float64 `json:"memory"`
	Goroutines int     `json:"goroutines"`
	GCPauses   float64 `json:"gcPauses"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Type      string `json:"type"`
	IP        string `json:"ip"`
	Message   string `json:"message"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	Status    int    `json:"status,omitempty"`
}

type UpstreamServer struct {
	Name      string  `json:"name"`
	URL       string  `json:"url"`
	Status    string  `json:"status"`
	Latency   float64 `json:"latency"`
	ErrorRate float64 `json:"errorRate"`
	Timeouts  int     `json:"timeouts"`
	LastCheck string  `json:"lastCheck"`
}

type RateLimitConfig struct {
	RPS            int `json:"rps"`
	RPM            int `json:"rpm"`
	MaxConnections int `json:"maxConnections"`
	BanDuration    int `json:"banDuration"`
}

type RateViolator struct {
	IP         string `json:"ip"`
	Requests   int64  `json:"requests"`
	Limit      int64  `json:"limit"`
	Violations int64  `json:"violations"`
}

func IPRuleFromDomain(r models.IPRule) IPRule {
	return IPRule{
		ID:        r.ID,
		IP:        r.IP,
		Type:      string(r.Type),
		Reason:    r.Reason,
		AddedDate: r.AddedDate.Format(time.RFC3339),
		Requests:  r.Requests,
	}
}

func IPAccessCheckFromDecision(ip string, allowed bool, reason string, grey bool) IPAccessCheck {
	return IPAccessCheck{
		IP:      ip,
		Allowed: allowed,
		Reason:  reason,
		Grey:    grey,
	}
}

func CaptchaVerifyFromDecision(ip string, allowed bool, reason string) CaptchaVerifyResult {
	return CaptchaVerifyResult{
		IP:      ip,
		Allowed: allowed,
		Reason:  reason,
	}
}

func ProxyMetricsFromDomain(m models.ProxyMetrics) ProxyMetrics {
	return ProxyMetrics{
		TotalRequests:     m.TotalRequests,
		RequestsPerSecond: m.RequestsPerSecond,
		AvgLatency:        m.AvgLatency,
		P99Latency:        m.P99Latency,
		ErrorRate:         m.ErrorRate,
		ActiveConnections: m.ActiveConnections,
	}
}

func CacheMetricsFromDomain(hits, misses, invalidations int64, size string) CacheMetrics {
	return CacheMetrics{Hits: hits, Misses: misses, Size: size, Invalidations: invalidations}
}

func SystemMetricsFromDomain(m models.SystemMetrics) SystemMetrics {
	return SystemMetrics{
		CPU:        m.CPU,
		Memory:     m.Memory,
		Goroutines: m.Goroutines,
		GCPauses:   m.GCPauses,
	}
}

func LogEntryFromDomain(e models.LogEntry) LogEntry {
	return LogEntry{
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Level:     string(e.Level),
		Type:      string(e.Type),
		IP:        e.IP,
		Message:   e.Message,
		Method:    e.Method,
		Path:      e.Path,
		Status:    e.Status,
	}
}

func UpstreamFromDomain(s models.UpstreamServer) UpstreamServer {
	return UpstreamServer{
		Name:      s.Name,
		URL:       s.URL,
		Status:    string(s.Status),
		Latency:   s.Latency,
		ErrorRate: s.ErrorRate,
		Timeouts:  s.Timeouts,
		LastCheck: s.LastCheck.Format(time.RFC3339),
	}
}

func RateLimitFromDomain(c models.RateLimitConfig) RateLimitConfig {
	return RateLimitConfig{
		RPS:            c.RPS,
		RPM:            c.RPM,
		MaxConnections: c.MaxConnections,
		BanDuration:    c.BanDuration,
	}
}

func RateLimitToDomain(c RateLimitConfig) models.RateLimitConfig {
	return models.RateLimitConfig{
		RPS:            c.RPS,
		RPM:            c.RPM,
		MaxConnections: c.MaxConnections,
		BanDuration:    c.BanDuration,
	}
}

func RateViolatorFromDomain(v models.RateViolator) RateViolator {
	return RateViolator{
		IP:         v.IP,
		Requests:   v.Requests,
		Limit:      v.Limit,
		Violations: v.Violations,
	}
}
