package handler

import (
	"net/http"
	"time"

	"GoProxy/internal/models"
	"GoProxy/internal/transport/http/v1/dto"
	"GoProxy/pkg/ratelimit"

	"github.com/gin-gonic/gin"
)

type RatelimiterHandler struct {
	limiter ratelimit.Limiter
}

func NewRatelimiterHandler(limiter ratelimit.Limiter) *RatelimiterHandler {
	return &RatelimiterHandler{limiter: limiter}
}

// @Summary Получить конфигурацию rate limit
// @Tags Admin
// @Produce json
// @Success 200 {object} dto.RateLimitConfigResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/rate-limit/config [get]
func (h *RatelimiterHandler) GetRateLimitConfig(c *gin.Context) {
	cfg := h.limiter.Config()
	c.JSON(http.StatusOK, dto.DataResponse[dto.RateLimitConfig]{
		Success: true,
		Data: dto.RateLimitFromDomain(models.RateLimitConfig{
			RPS:            cfg.RPS,
			RPM:            cfg.RPM,
			MaxConnections: cfg.MaxConnections,
			BanDuration:    int(cfg.BanDuration.Seconds()),
		}),
	})
}

// @Summary Обновить конфигурацию rate limit
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body dto.RateLimitConfig true "Конфигурация"
// @Success 200 {object} dto.RateLimitConfigResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/rate-limit/config [patch]
func (h *RatelimiterHandler) UpdateRateLimitConfig(c *gin.Context) {
	var req dto.RateLimitConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: err.Error()})
		return
	}

	domainCfg := dto.RateLimitToDomain(req)
	h.limiter.UpdateConfig(ratelimit.Config{
		RPS:            domainCfg.RPS,
		RPM:            domainCfg.RPM,
		MaxConnections: domainCfg.MaxConnections,
		BanDuration:    time.Duration(domainCfg.BanDuration) * time.Second,
	})

	c.JSON(http.StatusOK, dto.DataResponse[dto.RateLimitConfig]{Success: true, Data: req})
}

// @Summary Список нарушителей rate limit
// @Tags Admin
// @Produce json
// @Success 200 {object} dto.RateViolatorsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/rate-limit/violators [get]
func (h *RatelimiterHandler) GetRateViolators(c *gin.Context) {
	violators := h.limiter.Violators()
	out := make([]dto.RateViolator, len(violators))
	for i, v := range violators {
		out[i] = dto.RateViolator{
			IP:         v.IP,
			Requests:   v.Requests,
			Limit:      v.Limit,
			Violations: v.Violations,
		}
	}
	c.JSON(http.StatusOK, dto.DataResponse[[]dto.RateViolator]{Success: true, Data: out})
}
