// Package http содержит HTTP-слой сервиса: роутер, хендлеры и middleware.
package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Dasadno/testtask/internal/config"
)

// NewRouter собирает gin-роутер со всеми middleware и маршрутами.
func NewRouter(log *slog.Logger, env string) *gin.Engine {
	if env != config.EnvLocal {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(RequestLogger(log), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
