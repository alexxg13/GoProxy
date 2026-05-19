package main

import (
	"GoProxy/config"
	"GoProxy/internal/app"
	"flag"
	"fmt"
)

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
