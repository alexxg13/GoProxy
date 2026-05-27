package handler

import (
	"net/http"

	"GoProxy/internal/service/metrics"
	"GoProxy/internal/transport/http/v1/dto"
	"GoProxy/pkg/cache"

	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	collector metrics.Collector
	cache     cache.Cache
}

func NewMetricsHandler(collector metrics.Collector, cacheStore cache.Cache) *MetricsHandler {
	return &MetricsHandler{collector: collector, cache: cacheStore}
}

// @Summary Получить метрики прокси
// @Description RPS, latency, ошибки, активные соединения
// @Tags Admin
// @Produce json
// @Success 200 {object} dto.ProxyMetricsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/metrics/proxy [get]
func (h *MetricsHandler) GetProxyMetrics(c *gin.Context) {
	m := h.collector.ProxyMetrics()
	c.JSON(http.StatusOK, dto.DataResponse[dto.ProxyMetrics]{
		Success: true,
		Data:    dto.ProxyMetricsFromDomain(m),
	})
}

// @Summary Получить метрики кэша
// @Description Hits, misses, размер, инвалидации
// @Tags Admin
// @Produce json
// @Success 200 {object} dto.CacheMetricsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/metrics/cache [get]
func (h *MetricsHandler) GetCacheMetrics(c *gin.Context) {
	hits, misses, inv, size := h.cache.Metrics()
	c.JSON(http.StatusOK, dto.DataResponse[dto.CacheMetrics]{
		Success: true,
		Data:    dto.CacheMetricsFromDomain(hits, misses, inv, size),
	})
}

// @Summary Инвалидировать кэш вручную
// @Description Поддерживает режимы key, prefix, regex, tags и clear
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body dto.CacheInvalidationRequest true "Параметры инвалидации"
// @Success 200 {object} dto.CacheInvalidationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /admin/cache/invalidate [post]
func (h *MetricsHandler) InvalidateCache(c *gin.Context) {
	var req dto.CacheInvalidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: err.Error()})
		return
	}

	deleted := 0
	switch req.Mode {
	case "key":
		if req.Key == "" {
			c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: "key is required"})
			return
		}
		if h.cache.InvalidateKey(req.Key) {
			deleted = 1
		}
	case "prefix":
		if req.Prefix == "" {
			c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: "prefix is required"})
			return
		}
		deleted = h.cache.InvalidatePrefix(req.Prefix)
	case "regex":
		if req.Pattern == "" {
			c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: "pattern is required"})
			return
		}
		count, err := h.cache.InvalidateRegex(req.Pattern)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: err.Error()})
			return
		}
		deleted = count
	case "tags":
		if len(req.Tags) == 0 {
			c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: "tags are required"})
			return
		}
		deleted = h.cache.InvalidateTags(req.Tags)
	case "clear":
		h.cache.Clear()
	default:
		c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: "unsupported mode"})
		return
	}

	c.JSON(http.StatusOK, dto.DataResponse[dto.CacheInvalidationResult]{
		Success: true,
		Data:    dto.CacheInvalidationResult{Mode: req.Mode, Deleted: deleted},
	})
	h.collector.RecordCacheEvent("manual_" + req.Mode)
}

// @Summary Получить системные метрики
// @Description CPU, память, goroutines, GC
// @Tags Admin
// @Produce json
// @Success 200 {object} dto.SystemMetricsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/metrics/system [get]
func (h *MetricsHandler) GetSystemMetrics(c *gin.Context) {
	m := h.collector.SystemMetrics()
	c.JSON(http.StatusOK, dto.DataResponse[dto.SystemMetrics]{
		Success: true,
		Data:    dto.SystemMetricsFromDomain(m),
	})
}
