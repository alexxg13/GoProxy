package handler

import (
	"net/http"

	"GoProxy/internal/service/upstream"
	"GoProxy/internal/transport/http/v1/dto"

	"github.com/gin-gonic/gin"
)

type UpstreamHandler struct {
	checker upstream.Checker
}

func NewUpstreamHandler(checker upstream.Checker) *UpstreamHandler {
	return &UpstreamHandler{checker: checker}
}

// @Summary Статус upstream-серверов
// @Description Health, latency, error rate
// @Tags Admin
// @Produce json
// @Success 200 {object} dto.UpstreamServersResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/upstream-servers [get]
func (h *UpstreamHandler) GetUpstreamServers(c *gin.Context) {
	servers := h.checker.List()
	out := make([]dto.UpstreamServer, len(servers))
	for i, s := range servers {
		out[i] = dto.UpstreamFromDomain(s)
	}
	c.JSON(http.StatusOK, dto.DataResponse[[]dto.UpstreamServer]{Success: true, Data: out})
}
