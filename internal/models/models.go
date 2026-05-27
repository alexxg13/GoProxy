package models

import "time"

type RuleType string

const (
	RuleAllow RuleType = "allow"
	RuleDeny  RuleType = "deny"
	RuleGrey  RuleType = "grey"
)

type DenyReason string

const (
	DenyBlacklisted    DenyReason = "blacklisted"
	DenyNotWhitelisted DenyReason = "not_whitelisted"
	DenyGreyCaptcha    DenyReason = "grey_captcha_required"
	DenyRateLimited    DenyReason = "rate_limited"
)

type IPRule struct {
	ID        string
	IP        string
	Type      RuleType
	Reason    string
	AddedDate time.Time
	Requests  int64
}

type IPRuleCreate struct {
	IP     string
	Type   RuleType
	Reason string
}

type ProxyMetrics struct {
	TotalRequests     int64
	RequestsPerSecond float64
	AvgLatency        float64
	P99Latency        float64
	ErrorRate         float64
	ActiveConnections int64
}

type CacheMetrics struct {
	Hits          int64
	Misses        int64
	Size          string
	Invalidations int64
}

type SystemMetrics struct {
	CPU        float64
	Memory     float64
	Goroutines int
	GCPauses   float64
}

type LogLevel string

const (
	LogInfo    LogLevel = "info"
	LogWarning LogLevel = "warning"
	LogError   LogLevel = "error"
)

type LogType string

const (
	LogAccess    LogType = "access"
	LogErrorType LogType = "error"
	LogSecurity  LogType = "security"
)

type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Type      LogType
	IP        string
	Message   string
	Method    string
	Path      string
	Status    int
}

type UpstreamStatus string

const (
	UpstreamHealthy  UpstreamStatus = "healthy"
	UpstreamDegraded UpstreamStatus = "degraded"
	UpstreamDown     UpstreamStatus = "down"
)

type UpstreamServer struct {
	Name      string
	URL       string
	Status    UpstreamStatus
	Latency   float64
	ErrorRate float64
	Timeouts  int
	LastCheck time.Time
}

type RateLimitConfig struct {
	RPS            int `json:"rps"`
	RPM            int `json:"rpm"`
	MaxConnections int `json:"maxConnections"`
	BanDuration    int `json:"banDuration"`
}

type RateViolator struct {
	IP         string
	Requests   int64
	Limit      int64
	Violations int64
}
