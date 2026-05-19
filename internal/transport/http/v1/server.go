package v1

import (
	"GoProxy/config"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	srv      *http.Server
	engine   *gin.Engine
	handlers *Handlers
	proxy    *httputil.ReverseProxy
}

func NewServer(cfg config.Config, handlers *Handlers) *Server {
	if cfg.App.Env == "dev" {
		gin.SetMode(gin.DebugMode)
	} else if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	backendURL := cfg.Proxy.BackendURL
	url, _ := url.Parse(backendURL)
	proxy := httputil.NewSingleHostReverseProxy(url)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = url.Host
	}

	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}
	return &Server{
		srv:      &srv,
		engine:   engine,
		handlers: handlers,
	}
}

func (s *Server) Run() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) RegisterHandlers() {
	s.engine.GET("/_info", func(c *gin.Context) { c.String(200, "ok") })
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	protected := s.engine.Group("/admin")
	{
		metrics := protected.Group("/metrics")
		{
			metrics.GET("/proxy", s.handlers.MetricsHandler.GetProxyMetrics)
			metrics.GET("/cache", s.handlers.MetricsHandler.GetCacheMetrics)
			metrics.GET("/system", s.handlers.MetricsHandler.GetSystemMetrics)
		}

		ip := protected.Group("/ip-rules")
		{
			ip.GET("", s.handlers.IPHandler.GetIPRules)
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
