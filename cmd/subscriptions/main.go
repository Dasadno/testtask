// Сервис агрегации данных об онлайн-подписках пользователей.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/Dasadno/testtask/internal/app"
)

// @title       Subscriptions API
// @version     1.0
// @description REST-сервис для агрегации данных об онлайн-подписках пользователей.
// @BasePath    /api/v1
func main() {
	configPath := flag.String("config", envOrDefault("CONFIG_PATH", "config.yaml"), "path to config file")
	flag.Parse()

	if err := app.Run(*configPath); err != nil {
		log.Fatalf("service failed: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
