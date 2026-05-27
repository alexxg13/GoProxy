package middleware

import (
	"context"
	"fmt"
	"net/http"

	"GoProxy/internal/models"
	"GoProxy/internal/service/accesslog"

	"github.com/alexxg13/go-utils/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ginCtxKey struct{}

const lastUpstreamErrKey = "lastUpstreamError"

func AttachGinToRequest(c *gin.Context) {
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ginCtxKey{}, c))
}

func RecordUpstreamError(r *http.Request, err error) {
	if err == nil {
		return
	}
	c, ok := r.Context().Value(ginCtxKey{}).(*gin.Context)
	if !ok || c == nil {
		return
	}
	c.Set(lastUpstreamErrKey, err.Error())
}

func upstreamErrField(c *gin.Context) zap.Field {
	if msg, ok := c.Get(lastUpstreamErrKey); ok {
		if s, ok := msg.(string); ok && s != "" {
			return zap.String("upstream_error", s)
		}
	}
	return zap.Skip()
}

func writeAccessLog(ctx context.Context, zlog logger.Logger, store accesslog.Store, entry models.LogEntry, extra ...zap.Field) {
	store.Add(entry)

	fields := []zap.Field{
		zap.String("log_type", string(entry.Type)),
		zap.String("client_ip", entry.IP),
		zap.String("method", entry.Method),
		zap.String("path", entry.Path),
		zap.Int("status", entry.Status),
		zap.String("event_level", string(entry.Level)),
	}
	if entry.Message != "" {
		fields = append(fields, zap.String("reason", entry.Message))
	}
	fields = append(fields, extra...)

	zlog.Info(ctx, entry.Message, fields...)
}

func accessLogMessage(status int, cached bool, upstreamErr string) string {
	if cached {
		return "cache hit"
	}
	switch {
	case status >= 500:
		if upstreamErr != "" {
			return fmt.Sprintf("upstream error: %s", upstreamErr)
		}
		return "upstream error"
	case status >= 400:
		return "client error"
	default:
		return "request proxied"
	}
}
