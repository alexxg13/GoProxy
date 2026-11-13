package upstream

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"

	"GoProxy/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestChecker(status int, err error) *CheckerImpl {
	return &CheckerImpl{
		client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if err != nil {
				return nil, err
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(nil),
			}, nil
		})},
		servers: []models.UpstreamServer{{
			Name:   "backend",
			URL:    "http://backend.test",
			Status: models.UpstreamDown,
		}},
	}
}

func TestCheckerImpl_Healthy(t *testing.T) {
	checker := newTestChecker(http.StatusOK, nil)
	checker.check()

	servers := checker.List()
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	s := servers[0]
	if s.Status != models.UpstreamHealthy {
		t.Errorf("expected Healthy, got %v", s.Status)
	}
	if s.LastCheck.IsZero() {
		t.Error("LastCheck not set")
	}
}

func TestCheckerImpl_StatusDegraded(t *testing.T) {
	checker := newTestChecker(http.StatusInternalServerError, nil)
	checker.check()

	if got := checker.List()[0].Status; got != models.UpstreamDegraded {
		t.Errorf("expected Degraded, got %v", got)
	}
}

func TestCheckerImpl_StatusDownOnConnectionError(t *testing.T) {
	checker := newTestChecker(0, errors.New("connection refused"))
	checker.check()

	servers := checker.List()
	if servers[0].Status != models.UpstreamDown {
		t.Errorf("expected Down, got %v", servers[0].Status)
	}
	if servers[0].Timeouts != 1 {
		t.Errorf("expected Timeouts=1, got %d", servers[0].Timeouts)
	}
}

func TestCheckerImpl_InvalidURL(t *testing.T) {
	checker := newTestChecker(http.StatusOK, nil)
	checker.servers[0].URL = "://bad-url"
	checker.check()

	servers := checker.List()
	if servers[0].Status != models.UpstreamDown {
		t.Errorf("expected Down, got %v", servers[0].Status)
	}
	if servers[0].LastCheck.IsZero() {
		t.Error("LastCheck not set")
	}
}

func TestCheckerImpl_ListReturnsCopy(t *testing.T) {
	checker := newTestChecker(http.StatusOK, nil)
	checker.check()

	servers := checker.List()
	servers[0].Status = models.UpstreamDown

	if got := checker.List()[0].Status; got == models.UpstreamDown {
		t.Error("List returned slice that is not a copy; internal state mutated")
	}
}

func TestCheckerImpl_ConcurrentAccess(t *testing.T) {
	checker := newTestChecker(http.StatusOK, nil)
	checker.check()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = checker.List()
			}
		}()
	}
	wg.Wait()
}
