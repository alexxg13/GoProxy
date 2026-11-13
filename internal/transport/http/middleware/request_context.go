package middleware

import (
	"context"

	"github.com/alexxg13/go-utils/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const contextKey = "requestContext"

func RequestContext(base context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Header("X-Request-ID", reqID)

		reqCtx := logger.WithRequestID(base, reqID)
		if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
			reqCtx = logger.WithTraceID(reqCtx, traceID)
		}
		c.Set(contextKey, reqCtx)
		c.Next()
	}
}

func RequestCtx(c *gin.Context) context.Context {
	if v, ok := c.Get(contextKey); ok {
		if ctx, ok := v.(context.Context); ok {
			return ctx
		}
	}
	return c.Request.Context()
}
