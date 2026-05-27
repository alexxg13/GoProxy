package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeBackendURL(t *testing.T) {
	got, err := normalizeBackendURL("http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:3000" {
		t.Fatalf("expected 127.0.0.1, got %s", got)
	}
}

func TestNormalizeBackendURLDefaults(t *testing.T) {
	got, err := normalizeBackendURL("http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:80" {
		t.Fatalf("expected default http port, got %s", got)
	}

	got, err = normalizeBackendURL("https://localhost")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://127.0.0.1:443" {
		t.Fatalf("expected default https port, got %s", got)
	}
}

func TestParseConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	err := os.WriteFile(path, []byte(`APP_ENV=test
PORT=9090
READ_TIMEOUT=3s
WRITE_TIMEOUT=4s
LOG_LEVEL=debug
PROXY_BACKEND_URL=http://localhost:3000
IP_ACCESS_CONFIG=config/ip_rules.yaml
IP_DEFAULT_DENY=false
IP_CACHE_CAPACITY=12
IP_CACHE_TTL=2m
IP_CAPTCHA_TTL=5m
CACHE_DEFAULT_TTL=6m
CACHE_MAX_BODY=1234
RATE_LIMIT_RPS=7
RATE_LIMIT_RPM=8
RATE_LIMIT_MAX_CONN=9
RATE_LIMIT_BAN_SEC=10
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Port != 9090 || cfg.Proxy.BackendURL != "http://127.0.0.1:3000" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.IPAccess.CaptchaTTL != 5*time.Minute || cfg.RateLimit.BanDuration != 10 {
		t.Fatalf("unexpected durations: captcha=%s ban=%d", cfg.IPAccess.CaptchaTTL, cfg.RateLimit.BanDuration)
	}
}

func TestParseConfigError(t *testing.T) {
	if _, err := ParseConfig(filepath.Join(t.TempDir(), "missing.env")); err == nil {
		t.Fatal("expected missing config error")
	}
}
