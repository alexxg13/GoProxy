package accesslog

import (
	"sync"
	"time"

	"GoProxy/internal/models"
)

type Store interface {
	Add(entry models.LogEntry)
	List(level models.LogLevel, logType models.LogType, limit int) []models.LogEntry
}

type StoreImpl struct {
	mu      sync.RWMutex
	entries []models.LogEntry
	maxSize int
}

func NewStore(maxSize int) *StoreImpl {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &StoreImpl{maxSize: maxSize}
}

func (s *StoreImpl) Add(entry models.LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	s.entries = append(s.entries, entry)
	if len(s.entries) > s.maxSize {
		s.entries = s.entries[len(s.entries)-s.maxSize:]
	}
}

func (s *StoreImpl) List(level models.LogLevel, logType models.LogType, limit int) []models.LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	out := make([]models.LogEntry, 0, limit)
	for i := len(s.entries) - 1; i >= 0 && len(out) < limit; i-- {
		e := s.entries[i]
		if level != "" && e.Level != level {
			continue
		}
		if logType != "" && e.Type != logType {
			continue
		}
		out = append(out, e)
	}
	return out
}
