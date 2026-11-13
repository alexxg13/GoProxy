package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllow(t *testing.T) {
	l := New(Config{RPS: 100, RPM: 10000})
	if !l.Allow("1.2.3.4") {
		t.Fatal("first request should be allowed")
	}
}

func TestLimiterViolation(t *testing.T) {
	l := New(Config{RPS: 1, RPM: 100})
	ip := "203.0.113.1"
	l.Allow(ip)
	l.Allow(ip)
	l.Allow(ip)

	violators := l.Violators()
	if len(violators) == 0 {
		t.Fatal("expected violators")
	}
}

func TestLimiterHourAndDayViolation(t *testing.T) {
	for _, cfg := range []Config{
		{RPH: 1},
		{RPD: 1},
	} {
		l := New(cfg)
		ip := "203.0.113.2"
		if !l.Allow(ip) {
			t.Fatal("first request should be allowed")
		}
		if l.Allow(ip) {
			t.Fatalf("second request should be blocked for config %+v", cfg)
		}
	}
}

func TestLimiterUpdateConfig(t *testing.T) {
	l := New(Config{RPS: 10})
	l.UpdateConfig(Config{RPS: 200, RPH: 300, RPD: 400, BanDuration: time.Hour})
	cfg := l.Config()
	if cfg.RPS != 200 {
		t.Fatalf("expected 200, got %d", cfg.RPS)
	}
	if cfg.RPH != 300 || cfg.RPD != 400 {
		t.Fatalf("unexpected hour/day limits: %+v", cfg)
	}
}

func TestLimiterMaxConnections(t *testing.T) {
	l := New(Config{MaxConnections: 1})
	ip := "198.51.100.10"

	if !l.AllowConnection(ip) {
		t.Fatal("first connection should be allowed")
	}
	if l.AllowConnection(ip) {
		t.Fatal("second connection should be blocked")
	}

	l.ReleaseConnection(ip)
	if !l.AllowConnection(ip) {
		t.Fatal("connection should be allowed after release")
	}
}

func TestLimiterReleaseConnectionIsIdempotent(t *testing.T) {
	l := New(Config{MaxConnections: 1})
	ip := "198.51.100.11"

	l.ReleaseConnection(ip)
	if !l.AllowConnection(ip) {
		t.Fatal("release without acquire should not poison connection state")
	}
}
