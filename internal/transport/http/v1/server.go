package v1

import (
	"GoProxy/config"
	middleware2 "GoProxy/internal/transport/http/middleware"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	srv      *http.Server
	engine   *gin.Engine
	handlers *Handlers
	proxy    *httputil.ReverseProxy
}

func NewServer(cfg config.Config, handlers *Handlers, proxyMW *middleware2.ProxyMiddleware, baseCtx context.Context) *Server {
	if cfg.App.Env == "dev" {
		gin.SetMode(gin.DebugMode)
	} else if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware2.RequestContext(baseCtx))
	engine.Use(proxyMW.Handle())

	backendURL := cfg.Proxy.BackendURL
	target, err := url.Parse(backendURL)
	if err != nil {
		panic(fmt.Sprintf("invalid backend URL: %v", err))
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		middleware2.RecordUpstreamError(r, err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}

	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}
	s := &Server{
		srv:      &srv,
		engine:   engine,
		handlers: handlers,
		proxy:    proxy,
	}
	s.RegisterHandlers()
	return s
}

func (s *Server) Run() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) RegisterHandlers() {
	s.engine.GET("/_info", func(c *gin.Context) { c.String(200, "ok") })
	s.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	s.engine.POST("/captcha/verify", s.handlers.IPHandler.VerifyCaptcha)

	protected := s.engine.Group("/admin")
	{
		metrics := protected.Group("/metrics")
		{
			metrics.GET("/proxy", s.handlers.MetricsHandler.GetProxyMetrics)
			metrics.GET("/cache", s.handlers.MetricsHandler.GetCacheMetrics)
			metrics.GET("/system", s.handlers.MetricsHandler.GetSystemMetrics)
		}
		protected.POST("/cache/invalidate", s.handlers.MetricsHandler.InvalidateCache)

		ip := protected.Group("/ip-rules")
		{
			ip.GET("", s.handlers.IPHandler.GetIPRules)
			ip.GET("/check", s.handlers.IPHandler.CheckIPAccess)
			ip.POST("", s.handlers.IPHandler.CreateIPRule)
			ip.DELETE("/:ruleId", s.handlers.IPHandler.DeleteIPRule)
		}

		protected.GET("/logs", s.handlers.LogsHandler.GetLogs)
		protected.GET("/upstream-servers", s.handlers.UpstreamHandler.GetUpstreamServers)

		rate := protected.Group("/rate-limit")
		{
			rate.GET("/config", s.handlers.RatelimiterHandler.GetRateLimitConfig)
			rate.PATCH("/config", s.handlers.RatelimiterHandler.UpdateRateLimitConfig)
			rate.GET("/violators", s.handlers.RatelimiterHandler.GetRateViolators)
		}
	}

	s.engine.NoRoute(func(c *gin.Context) {
		s.proxy.ServeHTTP(c.Writer, c.Request)
	})
}
