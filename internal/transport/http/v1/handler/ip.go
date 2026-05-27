package handler

import (
	"net"
	"net/http"
	"strings"

	"GoProxy/internal/models"
	"GoProxy/internal/service/ipaccess"
	"GoProxy/internal/transport/http/v1/dto"

	"github.com/gin-gonic/gin"
)

type IPHandler struct {
	service ipaccess.IPService
}

func NewIPHandler(service ipaccess.IPService) *IPHandler {
	return &IPHandler{service: service}
}

// @Summary Получить IP-правила доступа
// @Description Белый, чёрный и серый списки
// @Tags Admin
// @Produce json
// @Param type query string false "Фильтр по типу" Enums(allow, deny, grey)
// @Success 200 {object} dto.IPRulesResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/ip-rules [get]
func (h *IPHandler) GetIPRules(c *gin.Context) {
	ruleType := models.RuleType(c.Query("type"))
	rules := h.service.List(ruleType)

	out := make([]dto.IPRule, len(rules))
	for i, r := range rules {
		out[i] = dto.IPRuleFromDomain(r)
	}

	c.JSON(http.StatusOK, dto.DataResponse[[]dto.IPRule]{Success: true, Data: out})
}

// @Summary Проверить доступ для IP
// @Description Возвращает итоговое решение с учётом deny/allow/grey/default policy
// @Tags Admin
// @Produce json
// @Param ip query string true "IP адрес"
// @Success 200 {object} dto.IPAccessCheckResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /admin/ip-rules/check [get]
func (h *IPHandler) CheckIPAccess(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: "ip query parameter is required"})
		return
	}

	decision := h.service.Check(c.Request.Context(), ip)
	c.JSON(http.StatusOK, dto.DataResponse[dto.IPAccessCheck]{
		Success: true,
		Data: dto.IPAccessCheckFromDecision(
			ip,
			decision.Allowed,
			string(decision.Reason),
			decision.Grey,
		),
	})
}

// @Summary Проверить CAPTCHA для серого списка
// @Description Статическая проверка 2 + 2. Верный ответ: 4
// @Tags Captcha
// @Accept json
// @Produce json
// @Param body body dto.CaptchaVerifyRequest true "Ответ на CAPTCHA"
// @Success 200 {object} dto.CaptchaVerifyResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.CaptchaVerifyResponse
// @Router /captcha/verify [post]
func (h *IPHandler) VerifyCaptcha(c *gin.Context) {
	var req dto.CaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: err.Error()})
		return
	}

	ip := requestIP(c.Request)
	decision := h.service.VerifyCaptcha(c.Request.Context(), ip, strings.TrimSpace(req.Answer))
	status := http.StatusOK
	if !decision.Allowed {
		status = http.StatusForbidden
	}

	c.JSON(status, dto.DataResponse[dto.CaptchaVerifyResult]{
		Success: decision.Allowed,
		Data: dto.CaptchaVerifyFromDecision(
			ip,
			decision.Allowed,
			string(decision.Reason),
		),
	})
}

// @Summary Добавить IP-правило
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body dto.IPRuleCreateRequest true "Новое правило"
// @Success 201 {object} dto.IPRuleResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/ip-rules [post]
func (h *IPHandler) CreateIPRule(c *gin.Context) {
	var req dto.IPRuleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: err.Error()})
		return
	}

	rule, err := h.service.Create(c.Request.Context(), models.IPRuleCreate{
		IP:     req.IP,
		Type:   models.RuleType(req.Type),
		Reason: req.Reason,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.SuccessResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.DataResponse[dto.IPRule]{
		Success: true,
		Data:    dto.IPRuleFromDomain(rule),
	})
}

// @Summary Удалить IP-правило
// @Tags Admin
// @Produce json
// @Param ruleId path string true "ID правила"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /admin/ip-rules/{ruleId} [delete]
func (h *IPHandler) DeleteIPRule(c *gin.Context) {
	id := c.Param("ruleId")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, dto.SuccessResponse{Success: false, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: true, Message: "IP rule deleted"})
}

func requestIP(r *http.Request) string {
	var raw string
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		raw = strings.TrimSpace(parts[0])
	} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
		raw = strings.TrimSpace(xri)
	} else {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			raw = r.RemoteAddr
		} else {
			raw = host
		}
	}
	return strings.TrimPrefix(strings.TrimSuffix(raw, "]"), "[")
}
