package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"GoProxy/internal/service/ipaccess"

	"github.com/gin-gonic/gin"
)

func TestIPHandlerGetRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccess.New(ipaccess.Config{DefaultDeny: true, CacheCapacity: 10, CacheTTL: time.Minute})
	h := NewIPHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/ip-rules", nil)

	h.GetIPRules(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestIPHandlerCreateRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccess.New(ipaccess.Config{DefaultDeny: true})
	h := NewIPHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"ip": "10.0.0.1", "type": "allow", "reason": "test",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/ip-rules", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateIPRule(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestIPHandlerCheckIPAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccess.New(ipaccess.Config{DefaultDeny: true})
	h := NewIPHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"ip": "10.0.0.1", "type": "allow", "reason": "test",
	})
	createCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/admin/ip-rules", bytes.NewReader(body))
	createCtx.Request.Header.Set("Content-Type", "application/json")
	h.CreateIPRule(createCtx)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/ip-rules/check?ip=10.0.0.1", nil)

	h.CheckIPAccess(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"allowed":true`) {
		t.Fatalf("expected allowed response, got %s", w.Body.String())
	}
}

func TestIPHandlerCheckIPAccessRequiresIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccess.New(ipaccess.Config{DefaultDeny: true})
	h := NewIPHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/ip-rules/check", nil)

	h.CheckIPAccess(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIPHandlerVerifyCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccess.New(ipaccess.Config{DefaultDeny: false, CaptchaTTL: time.Minute})
	h := NewIPHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"ip": "10.0.0.1", "type": "grey", "reason": "captcha",
	})
	createCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/admin/ip-rules", bytes.NewReader(body))
	createCtx.Request.Header.Set("Content-Type", "application/json")
	h.CreateIPRule(createCtx)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/captcha/verify", bytes.NewBufferString(`{"answer":"4"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Real-IP", "10.0.0.1")

	h.VerifyCaptcha(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"allowed":true`) {
		t.Fatalf("expected allowed response, got %s", w.Body.String())
	}
}

func TestIPHandlerVerifyCaptchaWrongAnswer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := ipaccess.New(ipaccess.Config{DefaultDeny: false, CaptchaTTL: time.Minute})
	h := NewIPHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"ip": "10.0.0.1", "type": "grey", "reason": "captcha",
	})
	createCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/admin/ip-rules", bytes.NewReader(body))
	createCtx.Request.Header.Set("Content-Type", "application/json")
	h.CreateIPRule(createCtx)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/captcha/verify", bytes.NewBufferString(`{"answer":"5"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Real-IP", "10.0.0.1")

	h.VerifyCaptcha(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}
