package ratelimit

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
)

type Violation struct {
	IP         string
	Requests   int64
	Limit      int64
	Violations int64
	LastSeen   time.Time
}

type Config struct {
	RPS            int
	RPM            int
	MaxConnections int
	BanDuration    time.Duration
}
type Limiter interface {
	UpdateConfig(cfg Config)
	Config() Config
	AllowConnection(ip string) bool
	ReleaseConnection(ip string)
	Allow(ip string) bool
	Violators() []Violation
}
type LimiterImpl struct {
	mu          sync.RWMutex
	cfg         Config
	limiters    map[string]*clientLimiter
	connections map[string]int
	violators   map[string]*Violation
}

type clientLimiter struct {
	perSecond *rate.Limiter
	perMinute *rate.Limiter
	requests  int64
}

var rateLimitConfigGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "goproxy_rate_limit_config",
	Help: "Current rate limit configuration values.",
}, []string{"limit"})

func New(cfg Config) *LimiterImpl {
	setRateLimitConfigMetrics(cfg)
	return &LimiterImpl{
		cfg:         cfg,
		limiters:    make(map[string]*clientLimiter),
		connections: make(map[string]int),
		violators:   make(map[string]*Violation),
	}
}

func (l *LimiterImpl) UpdateConfig(cfg Config) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cfg = cfg
	setRateLimitConfigMetrics(cfg)
}

func (l *LimiterImpl) Config() Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cfg
}

func (l *LimiterImpl) AllowConnection(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cfg.MaxConnections > 0 && l.connections[ip] >= l.cfg.MaxConnections {
		l.recordViolation(ip, int64(l.cfg.MaxConnections))
		return false
	}
	l.connections[ip]++
	return true
}

func (l *LimiterImpl) ReleaseConnection(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.connections[ip] <= 1 {
		delete(l.connections, ip)
		return
	}
	l.connections[ip]--
}

func (l *LimiterImpl) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cl, ok := l.limiters[ip]
	if !ok {
		cl = l.newClientLimiter()
		l.limiters[ip] = cl
	}

	cl.requests++

	if l.cfg.RPS > 0 && !cl.perSecond.Allow() {
		l.recordViolation(ip, int64(l.cfg.RPS))
		return false
	}
	if l.cfg.RPM > 0 && !cl.perMinute.Allow() {
		l.recordViolation(ip, int64(l.cfg.RPM))
		return false
	}

	return true
}

func (l *LimiterImpl) Violators() []Violation {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]Violation, 0, len(l.violators))
	for _, v := range l.violators {
		out = append(out, *v)
	}
	return out
}

func (l *LimiterImpl) newClientLimiter() *clientLimiter {
	cl := &clientLimiter{}
	if l.cfg.RPS > 0 {
		cl.perSecond = rate.NewLimiter(rate.Limit(l.cfg.RPS), l.cfg.RPS)
	} else {
		cl.perSecond = rate.NewLimiter(rate.Inf, 0)
	}
	if l.cfg.RPM > 0 {
		cl.perMinute = rate.NewLimiter(rate.Every(time.Minute/time.Duration(l.cfg.RPM)), l.cfg.RPM)
	} else {
		cl.perMinute = rate.NewLimiter(rate.Inf, 0)
	}
	return cl
}

func (l *LimiterImpl) recordViolation(ip string, limit int64) {
	v, ok := l.violators[ip]
	if !ok {
		v = &Violation{IP: ip, Limit: limit}
		l.violators[ip] = v
	}
	v.Violations++
	v.Requests++
	v.LastSeen = time.Now()
}

func setRateLimitConfigMetrics(cfg Config) {
	rateLimitConfigGauge.WithLabelValues("rps").Set(float64(cfg.RPS))
	rateLimitConfigGauge.WithLabelValues("rpm").Set(float64(cfg.RPM))
	rateLimitConfigGauge.WithLabelValues("max_connections").Set(float64(cfg.MaxConnections))
	rateLimitConfigGauge.WithLabelValues("ban_seconds").Set(cfg.BanDuration.Seconds())
}
