package main

import (
	"GoProxy/config"
	_ "GoProxy/docs"
	"GoProxy/internal/app"
	"flag"
	"fmt"
)

// @title GoProxy Admin API
// @version 1.0
// @description REST API админ-панели reverse-proxy: метрики, IP-правила, логи, rate limit, upstream.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@goproxy.local

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @schemes http https

// @tag.name Admin
// @tag.description Административные операции прокси-сервера
func main() {
	envPath := flag.String("env", ".env", ".env")
	flag.Parse()
	cfg, err := config.ParseConfig(*envPath)
	if err != nil {
		fmt.Printf("failed to parse config: %v\n", err)
		return
	}
	app.Run(cfg)
}
