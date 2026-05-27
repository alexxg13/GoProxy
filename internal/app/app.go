package app

import (
	"GoProxy/config"
	"GoProxy/internal/service/accesslog"
	"GoProxy/internal/service/ipaccess"
	"GoProxy/internal/service/metrics"
	"GoProxy/internal/service/upstream"
	"GoProxy/internal/transport/http/middleware"
	v1 "GoProxy/internal/transport/http/v1"
	"GoProxy/internal/transport/http/v1/handler"
	"GoProxy/pkg/cache"
	"GoProxy/pkg/ratelimit"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexxg13/go-utils/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func Run(cfg *config.Config) {
	ctx := context.Background()

	ctx = context.WithValue(ctx, logger.LoggerEnvKey, cfg.Log.Level)
	ctx = logger.WithRequestID(ctx, uuid.NewString())
	ctx = logger.WithTraceID(ctx, uuid.NewString())

	ctx, err := logger.New(ctx)
	if err != nil {
		fmt.Printf("failed to create logger: %v\n", err)
		return
	}

	collector := metrics.NewCollector()
	cacheStore := cache.NewStore(cache.Config{
		DefaultTTL:  cfg.Cache.DefaultTTL,
		MaxBodySize: cfg.Cache.MaxBodySize,
	})

	ipService, err := ipaccess.New(ipaccess.Config{
		ConfigPath:    cfg.IPAccess.ConfigPath,
		CacheCapacity: cfg.IPAccess.CacheCapacity,
		CacheTTL:      cfg.IPAccess.CacheTTL,
		CaptchaTTL:    cfg.IPAccess.CaptchaTTL,
		DefaultDeny:   cfg.IPAccess.DefaultDeny,
	})
	if err != nil {
		logger.GetLogger(ctx).Error(ctx, "failed to init ip access service", zap.Error(err))
		return
	}

	limiter := ratelimit.New(ratelimit.Config{
		RPS:            cfg.RateLimit.RPS,
		RPM:            cfg.RateLimit.RPM,
		MaxConnections: cfg.RateLimit.MaxConnections,
		BanDuration:    time.Duration(cfg.RateLimit.BanDuration) * time.Second,
	})

	logStore := accesslog.NewStore(10000)
	upstreamChecker := upstream.NewChecker(cfg.Proxy.BackendURL)

	handlers := v1.NewHandlers(
		handler.NewMetricsHandler(collector, cacheStore),
		handler.NewIPHandler(ipService),
		handler.NewLogsHandler(logStore),
		handler.NewUpstreamHandler(upstreamChecker),
		handler.NewRatelimiterHandler(limiter),
	)

	proxyMW := middleware.NewProxyMiddleware(ipService, limiter, collector, logStore, logger.GetLogger(ctx), cacheStore)
	server := v1.NewServer(*cfg, handlers, proxyMW, ctx)

	go func() {
		logger.GetLogger(ctx).Info(ctx, "starting proxy server", zap.Int("port", cfg.HTTP.Port))
		if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.GetLogger(ctx).Error(ctx, "server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.GetLogger(ctx).Info(ctx, "shutting down proxy server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.GetLogger(ctx).Error(ctx, "shutdown error", zap.Error(err))
	}
	logger.GetLogger(ctx).Info(ctx, "server stopped")
}
