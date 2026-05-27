package accesslog

import (
	"testing"

	"GoProxy/internal/models"
)

func TestStoreAddList(t *testing.T) {
	s := NewStore(10)
	s.Add(models.LogEntry{
		Level: models.LogWarning, Type: models.LogSecurity,
		IP: "1.2.3.4", Message: "blacklisted", Method: "GET", Path: "/api", Status: 403,
	})

	entries := s.List(models.LogWarning, models.LogSecurity, 10)
	if len(entries) != 1 || entries[0].IP != "1.2.3.4" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}
