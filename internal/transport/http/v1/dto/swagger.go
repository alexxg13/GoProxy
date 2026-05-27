package dto

type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"invalid request"`
}

type ProxyMetricsResponse struct {
	Success bool         `json:"success" example:"true"`
	Data    ProxyMetrics `json:"data"`
}

type CacheMetricsResponse struct {
	Success bool         `json:"success" example:"true"`
	Data    CacheMetrics `json:"data"`
}

type SystemMetricsResponse struct {
	Success bool          `json:"success" example:"true"`
	Data    SystemMetrics `json:"data"`
}

type IPRulesResponse struct {
	Success bool     `json:"success" example:"true"`
	Data    []IPRule `json:"data"`
}

type IPRuleResponse struct {
	Success bool   `json:"success" example:"true"`
	Data    IPRule `json:"data"`
}

type IPAccessCheckResponse struct {
	Success bool          `json:"success" example:"true"`
	Data    IPAccessCheck `json:"data"`
}

type CaptchaVerifyResponse struct {
	Success bool                `json:"success" example:"true"`
	Data    CaptchaVerifyResult `json:"data"`
}

type CacheInvalidationResponse struct {
	Success bool                    `json:"success" example:"true"`
	Data    CacheInvalidationResult `json:"data"`
}

type LogsResponse struct {
	Success bool       `json:"success" example:"true"`
	Data    []LogEntry `json:"data"`
}

type UpstreamServersResponse struct {
	Success bool             `json:"success" example:"true"`
	Data    []UpstreamServer `json:"data"`
}

type RateLimitConfigResponse struct {
	Success bool            `json:"success" example:"true"`
	Data    RateLimitConfig `json:"data"`
}

type RateViolatorsResponse struct {
	Success bool           `json:"success" example:"true"`
	Data    []RateViolator `json:"data"`
}
