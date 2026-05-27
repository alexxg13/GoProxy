package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	ExpiresAt  time.Time
	Tags       []string
}
type Cache interface {
	Key(method, host, path string) string
	Get(key string) (*Entry, bool)
	Set(key string, entry *Entry)
	InvalidateKey(key string) bool
	InvalidatePrefix(prefix string) int
	InvalidateRegex(pattern string) (int, error)
	InvalidateTags(tags []string) int
	Clear()
	Metrics() (hits, misses, invalidations int64, size string)
}
type CacheImpl struct {
	mu            sync.RWMutex
	items         map[string]*Entry
	defaultTTL    time.Duration
	maxBodySize   int
	hits          int64
	misses        int64
	invalidations int64
}

type Config struct {
	DefaultTTL  time.Duration
	MaxBodySize int
}

func NewStore(cfg Config) *CacheImpl {
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = 1 << 20
	}
	return &CacheImpl{
		items:       make(map[string]*Entry),
		defaultTTL:  cfg.DefaultTTL,
		maxBodySize: cfg.MaxBodySize,
	}
}

func (s *CacheImpl) Key(method, host, path string) string {
	h := sha256.Sum256([]byte(method + "|" + host + "|" + path))
	return hex.EncodeToString(h[:])
}

func (s *CacheImpl) Get(key string) (*Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.items[key]
	if !ok {
		s.misses++
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(s.items, key)
		s.misses++
		return nil, false
	}
	s.hits++
	return entry, true
}

func (s *CacheImpl) Set(key string, entry *Entry) {
	if len(entry.Body) > s.maxBodySize {
		return
	}
	if entry.ExpiresAt.IsZero() {
		entry.ExpiresAt = time.Now().Add(s.defaultTTL)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = entry
}

func (s *CacheImpl) InvalidateKey(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[key]; ok {
		delete(s.items, key)
		s.invalidations++
		return true
	}
	return false
}

func (s *CacheImpl) InvalidatePrefix(prefix string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for key := range s.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.items, key)
			count++
		}
	}
	s.invalidations += int64(count)
	return count
}

func (s *CacheImpl) InvalidateRegex(pattern string) (int, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for key := range s.items {
		if re.MatchString(key) {
			delete(s.items, key)
			count++
		}
	}
	s.invalidations += int64(count)
	return count, nil
}

func (s *CacheImpl) InvalidateTags(tags []string) int {
	if len(tags) == 0 {
		return 0
	}
	lookup := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			lookup[trimmed] = struct{}{}
		}
	}
	if len(lookup) == 0 {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for key, entry := range s.items {
		for _, tag := range entry.Tags {
			if _, ok := lookup[tag]; ok {
				delete(s.items, key)
				count++
				break
			}
		}
	}
	s.invalidations += int64(count)
	return count
}

func (s *CacheImpl) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.items)
	s.items = make(map[string]*Entry)
	s.invalidations += int64(n)
}

func (s *CacheImpl) Metrics() (hits, misses, invalidations int64, size string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var bytes int
	for _, e := range s.items {
		bytes += len(e.Body)
	}
	return s.hits, s.misses, s.invalidations, formatSize(bytes)
}

func formatSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := int64(bytes) / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func ShouldCache(status int, cacheControl string) bool {
	for _, directive := range strings.Split(cacheControl, ",") {
		switch strings.ToLower(strings.TrimSpace(directive)) {
		case "no-cache", "no-store", "private":
			return false
		}
	}
	return status >= 200 && status < 300
}
