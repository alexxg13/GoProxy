package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		App   App
		HTTP  HTTP
		Log   Log
		DB    DB
		JWT   JWT
		Proxy Proxy
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
	DB struct {
		Host     string `env:"POSTGRES_HOST" envDefault:"db"`
		Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
		Username string `env:"POSTGRES_USER" envDefault:"postgres"`
		Password string `env:"POSTGRES_PASSWORD" envDefault:"postgres"`
		Database string `env:"POSTGRES_DB" envDefault:"DB"`
	}
	JWT struct {
		Secret string `env:"JWT_SECRET" envDefault:"secret"`
	}
	Proxy struct {
		BackendURL string `env:"PROXY_BACKEND_URL" envDefault:"http://localhost:8080"`
	}
)

func ParseConfig(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}
