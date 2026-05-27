package v1

import "github.com/gin-gonic/gin"

type (
	Handlers struct {
		MetricsHandler     MetricsHandler
		IPHandler          IPHandler
		LogsHandler        LogsHandler
		UpstreamHandler    UpstreamHandler
		RatelimiterHandler RatelimiterHandler
	}

	MetricsHandler interface {
		GetProxyMetrics(c *gin.Context)
		GetCacheMetrics(c *gin.Context)
		InvalidateCache(c *gin.Context)
		GetSystemMetrics(c *gin.Context)
	}

	IPHandler interface {
		GetIPRules(c *gin.Context)
		CheckIPAccess(c *gin.Context)
		VerifyCaptcha(c *gin.Context)
		CreateIPRule(c *gin.Context)
		DeleteIPRule(c *gin.Context)
	}

	LogsHandler interface {
		GetLogs(c *gin.Context)
	}

	UpstreamHandler interface {
		GetUpstreamServers(c *gin.Context)
	}

	RatelimiterHandler interface {
		GetRateLimitConfig(c *gin.Context)
		UpdateRateLimitConfig(c *gin.Context)
		GetRateViolators(c *gin.Context)
	}
)

func NewHandlers(metricsHandler MetricsHandler, ipHandler IPHandler, logsHandler LogsHandler, upstreamHandler UpstreamHandler, ratelimiterHandler RatelimiterHandler) *Handlers {
	return &Handlers{
		MetricsHandler:     metricsHandler,
		IPHandler:          ipHandler,
		LogsHandler:        logsHandler,
		UpstreamHandler:    upstreamHandler,
		RatelimiterHandler: ratelimiterHandler,
	}

}
