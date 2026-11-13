package handler

import (
	"net/http"
	"strconv"

	"GoProxy/internal/models"
	"GoProxy/internal/service/accesslog"
	"GoProxy/internal/transport/http/v1/dto"

	"github.com/gin-gonic/gin"
)

type LogsHandler struct {
	store accesslog.Store
}

func NewLogsHandler(store accesslog.Store) *LogsHandler {
	return &LogsHandler{store: store}
}

// @Summary Получить логи доступа
// @Description Буфер последних событий; полный архив — в logs/app.json (go-utils/zap)
// @Tags Admin
// @Produce json
// @Param level query string false "Уровень" Enums(info, warning, error)
// @Param type query string false "Тип" Enums(access, error, security)
// @Param limit query int false "Количество записей" default(100)
// @Success 200 {object} dto.LogsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/logs [get]
func (h *LogsHandler) GetLogs(c *gin.Context) {
	level := models.LogLevel(c.Query("level"))
	logType := models.LogType(c.Query("type"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	entries := h.store.List(level, logType, limit)
	out := make([]dto.LogEntry, len(entries))
	for i, e := range entries {
		out[i] = dto.LogEntryFromDomain(e)
	}

	c.JSON(http.StatusOK, dto.DataResponse[[]dto.LogEntry]{Success: true, Data: out})
}
