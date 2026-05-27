package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App       App
		HTTP      HTTP
		Log       Log
		Proxy     Proxy
		IPAccess  IPAccess
		Cache     Cache
		RateLimit RateLimit
	}
	App struct {
		Env string `env:"APP_ENV" envDefault:"dev"`
	}
	HTTP struct {
		Port         int           `env:"PORT" envDefault:"8080"`
		ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
		WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	}
	Log struct {
		Level string `env:"LOG_LEVEL" envDefault:"info"`
	}
	Proxy struct {
		BackendURL string `env:"PROXY_BACKEND_URL" envDefault:"http://127.0.0.1:3000"`
	}
	IPAccess struct {
		ConfigPath    string        `env:"IP_ACCESS_CONFIG" envDefault:"config/ip_rules.yaml"`
		DefaultDeny   bool          `env:"IP_DEFAULT_DENY" envDefault:"true"`
		CacheCapacity int           `env:"IP_CACHE_CAPACITY" envDefault:"4096"`
		CacheTTL      time.Duration `env:"IP_CACHE_TTL" envDefault:"1m"`
		CaptchaTTL    time.Duration `env:"IP_CAPTCHA_TTL" envDefault:"10m"`
	}
	Cache struct {
		DefaultTTL  time.Duration `env:"CACHE_DEFAULT_TTL" envDefault:"5m"`
		MaxBodySize int           `env:"CACHE_MAX_BODY" envDefault:"1048576"`
	}
	RateLimit struct {
		RPS            int `env:"RATE_LIMIT_RPS" envDefault:"100"`
		RPM            int `env:"RATE_LIMIT_RPM" envDefault:"1000"`
		MaxConnections int `env:"RATE_LIMIT_MAX_CONN" envDefault:"500"`
		BanDuration    int `env:"RATE_LIMIT_BAN_SEC" envDefault:"3600"`
	}
)

func ParseConfig(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	normalized, err := normalizeBackendURL(cfg.Proxy.BackendURL)
	if err != nil {
		return nil, err
	}
	cfg.Proxy.BackendURL = normalized

	return cfg, nil
}

func normalizeBackendURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse backend URL: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}

	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		port := u.Port()
		if port == "" {
			port = "80"
			if u.Scheme == "https" {
				port = "443"
			}
		}
		u.Host = net.JoinHostPort("127.0.0.1", port)
	}

	return u.String(), nil
}
