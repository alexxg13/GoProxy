package lrucache

import (
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := New[string, int](2, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)

	v, ok := c.Get("a")
	if !ok || v != 1 {
		t.Fatalf("expected 1, got %d ok=%v", v, ok)
	}
}

func TestCacheEviction(t *testing.T) {
	c := New[string, int](2, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	if _, ok := c.Get("a"); ok {
		t.Fatal("expected a to be evicted")
	}
}

func TestCacheTTL(t *testing.T) {
	c := New[string, int](10, 50*time.Millisecond)
	c.Set("a", 1)
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected expired entry")
	}
}

func TestCacheDeleteLenAndDefaultCapacity(t *testing.T) {
	c := New[string, int](0, time.Minute)
	c.Set("a", 1)
	if c.Len() != 1 {
		t.Fatalf("zero capacity should use default capacity, len=%d", c.Len())
	}

	c = New[string, int](2, time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	if c.Len() != 2 {
		t.Fatalf("expected len 2, got %d", c.Len())
	}
	c.Delete("a")
	if c.Len() != 1 {
		t.Fatalf("expected len 1, got %d", c.Len())
	}
	if _, ok := c.Get("a"); ok {
		t.Fatal("deleted key should miss")
	}
	c.Delete("missing")
	if c.Len() != 1 {
		t.Fatalf("delete missing should not change len, got %d", c.Len())
	}
}

func TestCacheSetUpdatesExisting(t *testing.T) {
	c := New[string, int](2, time.Minute)
	c.Set("a", 1)
	c.Set("a", 2)

	v, ok := c.Get("a")
	if !ok || v != 2 {
		t.Fatalf("expected updated value 2, got %d ok=%v", v, ok)
	}
	if c.Len() != 1 {
		t.Fatalf("expected len 1, got %d", c.Len())
	}
}
