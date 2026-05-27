package upstream

import (
	"context"
	"net/http"
	"sync"
	"time"

	"GoProxy/internal/models"
)

type Checker interface {
	List() []models.UpstreamServer
}

type CheckerImpl struct {
	mu      sync.RWMutex
	servers []models.UpstreamServer
	client  *http.Client
}

func NewChecker(backendURL string) *CheckerImpl {
	c := &CheckerImpl{
		client: &http.Client{Timeout: 5 * time.Second},
		servers: []models.UpstreamServer{{
			Name:   "backend",
			URL:    backendURL,
			Status: models.UpstreamDown,
		}},
	}
	go c.runLoop()
	return c
}

func (c *CheckerImpl) runLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	c.check()
	for range ticker.C {
		c.check()
	}
}

func (c *CheckerImpl) check() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.servers {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.servers[i].URL, nil)
		if err != nil {
			cancel()
			c.servers[i].Status = models.UpstreamDown
			c.servers[i].LastCheck = time.Now().UTC()
			continue
		}

		resp, err := c.client.Do(req)
		cancel()
		latency := time.Since(start).Seconds() * 1000

		c.servers[i].Latency = latency
		c.servers[i].LastCheck = time.Now().UTC()

		if err != nil {
			c.servers[i].Status = models.UpstreamDown
			c.servers[i].Timeouts++
			continue
		}
		resp.Body.Close()

		switch {
		case resp.StatusCode >= 500:
			c.servers[i].Status = models.UpstreamDegraded
		default:
			c.servers[i].Status = models.UpstreamHealthy
		}
	}
}

func (c *CheckerImpl) List() []models.UpstreamServer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]models.UpstreamServer(nil), c.servers...)
}
