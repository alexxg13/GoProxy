package dto

import (
	"testing"
	"time"

	"GoProxy/internal/models"
)

func TestDomainConverters(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)

	ip := IPRuleFromDomain(models.IPRule{
		ID: "1", IP: "10.0.0.1", Type: models.RuleAllow, Reason: "test", AddedDate: now, Requests: 7,
	})
	if ip.ID != "1" || ip.Type != "allow" || ip.Requests != 7 || ip.AddedDate == "" {
		t.Fatalf("bad IP rule conversion: %+v", ip)
	}

	check := IPAccessCheckFromDecision("1.2.3.4", true, "", false)
	if !check.Allowed || check.IP != "1.2.3.4" {
		t.Fatalf("bad access check conversion: %+v", check)
	}

	captcha := CaptchaVerifyFromDecision("1.2.3.4", false, "grey_captcha_required")
	if captcha.Allowed || captcha.Reason == "" {
		t.Fatalf("bad captcha conversion: %+v", captcha)
	}

	proxy := ProxyMetricsFromDomain(models.ProxyMetrics{
		TotalRequests: 10, RequestsPerSecond: 2, AvgLatency: 3, P99Latency: 4, ErrorRate: 5, ActiveConnections: 6,
	})
	if proxy.TotalRequests != 10 || proxy.ActiveConnections != 6 {
		t.Fatalf("bad proxy metrics conversion: %+v", proxy)
	}

	cache := CacheMetricsFromDomain(1, 2, 3, "4B")
	if cache.Hits != 1 || cache.Misses != 2 || cache.Invalidations != 3 || cache.Size != "4B" {
		t.Fatalf("bad cache metrics conversion: %+v", cache)
	}

	system := SystemMetricsFromDomain(models.SystemMetrics{CPU: 1, Memory: 2, Goroutines: 3, GCPauses: 4})
	if system.CPU != 1 || system.Goroutines != 3 {
		t.Fatalf("bad system metrics conversion: %+v", system)
	}

	log := LogEntryFromDomain(models.LogEntry{
		Timestamp: now, Level: models.LogInfo, Type: models.LogAccess, IP: "127.0.0.1", Message: "ok", Method: "GET", Path: "/", Status: 200,
	})
	if log.Timestamp == "" || log.Level != "info" || log.Status != 200 {
		t.Fatalf("bad log conversion: %+v", log)
	}

	upstream := UpstreamFromDomain(models.UpstreamServer{
		Name: "backend", URL: "http://backend", Status: models.UpstreamHealthy, Latency: 1.5, ErrorRate: 2.5, Timeouts: 1, LastCheck: now,
	})
	if upstream.Name != "backend" || upstream.Status != "healthy" || upstream.LastCheck == "" {
		t.Fatalf("bad upstream conversion: %+v", upstream)
	}

	rate := RateLimitFromDomain(models.RateLimitConfig{RPS: 1, RPM: 2, MaxConnections: 3, BanDuration: 4})
	if rate.RPS != 1 || rate.BanDuration != 4 {
		t.Fatalf("bad rate limit conversion: %+v", rate)
	}

	domainRate := RateLimitToDomain(rate)
	if domainRate.RPM != 2 || domainRate.MaxConnections != 3 {
		t.Fatalf("bad domain rate limit conversion: %+v", domainRate)
	}

	violator := RateViolatorFromDomain(models.RateViolator{IP: "1.2.3.4", Requests: 1, Limit: 2, Violations: 3})
	if violator.IP != "1.2.3.4" || violator.Violations != 3 {
		t.Fatalf("bad violator conversion: %+v", violator)
	}
}
