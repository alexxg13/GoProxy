package cache

import (
	"net/http"
	"testing"
	"time"
)

func TestStoreGetSet(t *testing.T) {
	s := NewStore(Config{DefaultTTL: time.Minute})
	key := s.Key("GET", "example.com", "/api")
	s.Set(key, &Entry{StatusCode: 200, Body: []byte("ok"), Headers: http.Header{}})

	entry, ok := s.Get(key)
	if !ok || string(entry.Body) != "ok" {
		t.Fatal("expected cache hit")
	}
}

func TestStoreExpiredEntryAndMissMetrics(t *testing.T) {
	s := NewStore(Config{DefaultTTL: time.Millisecond})
	s.Set("expired", &Entry{Body: []byte("old"), ExpiresAt: time.Now().Add(-time.Second)})

	if _, ok := s.Get("missing"); ok {
		t.Fatal("missing key should not hit")
	}
	if _, ok := s.Get("expired"); ok {
		t.Fatal("expired key should not hit")
	}

	_, misses, _, _ := s.Metrics()
	if misses != 2 {
		t.Fatalf("expected 2 misses, got %d", misses)
	}
}

func TestShouldCache(t *testing.T) {
	if !ShouldCache(200, "") {
		t.Fatal("2xx should be cached")
	}
	if ShouldCache(404, "") {
		t.Fatal("4xx should not be cached by default")
	}
	if ShouldCache(200, "no-cache") {
		t.Fatal("no-cache should skip")
	}
	if ShouldCache(200, "public, max-age=60, private") {
		t.Fatal("private directive should skip")
	}
}

func TestInvalidateKeyClearMetricsAndFormatSize(t *testing.T) {
	s := NewStore(Config{})
	s.Set("a", &Entry{Body: []byte("1")})
	s.Set("b", &Entry{Body: make([]byte, 2048)})

	if !s.InvalidateKey("a") {
		t.Fatal("expected key invalidation")
	}
	if s.InvalidateKey("missing") {
		t.Fatal("missing key should not invalidate")
	}
	_, _, invalidations, size := s.Metrics()
	if invalidations != 1 || size == "0B" {
		t.Fatalf("unexpected metrics invalidations=%d size=%s", invalidations, size)
	}

	s.Clear()
	_, _, invalidations, size = s.Metrics()
	if invalidations != 2 || size != "0B" {
		t.Fatalf("unexpected metrics after clear invalidations=%d size=%s", invalidations, size)
	}
}

func TestInvalidatePrefix(t *testing.T) {
	s := NewStore(Config{})
	s.Set("abc123", &Entry{Body: []byte("1")})
	s.Set("abc456", &Entry{Body: []byte("2")})
	n := s.InvalidatePrefix("abc")
	if n != 2 {
		t.Fatalf("expected 2 invalidations, got %d", n)
	}
}

func TestInvalidateRegex(t *testing.T) {
	s := NewStore(Config{})
	s.Set("user:1", &Entry{Body: []byte("1")})
	s.Set("post:1", &Entry{Body: []byte("2")})

	n, err := s.InvalidateRegex(`^user:`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 invalidation, got %d", n)
	}
	if _, ok := s.Get("user:1"); ok {
		t.Fatal("user key should be invalidated")
	}
}

func TestInvalidateTags(t *testing.T) {
	s := NewStore(Config{})
	s.Set("a", &Entry{Body: []byte("1"), Tags: []string{"catalog", "hot"}})
	s.Set("b", &Entry{Body: []byte("2"), Tags: []string{"profile"}})

	n := s.InvalidateTags([]string{"catalog"})
	if n != 1 {
		t.Fatalf("expected 1 invalidation, got %d", n)
	}
	if _, ok := s.Get("a"); ok {
		t.Fatal("tagged entry should be invalidated")
	}
	if _, ok := s.Get("b"); !ok {
		t.Fatal("unmatched tagged entry should remain")
	}
}
