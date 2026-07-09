// Package http содержит HTTP-слой сервиса: роутер, хендлеры и middleware.
package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	// Регистрация сгенерированной swagger-спецификации.
	_ "github.com/Dasadno/testtask/api"
	"github.com/Dasadno/testtask/internal/config"
)

// NewRouter собирает gin-роутер со всеми middleware и маршрутами.
func NewRouter(log *slog.Logger, env string, subscriptions *SubscriptionHandler) *gin.Engine {
	if env != config.EnvLocal {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(RequestLogger(log), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	api := router.Group("/api/v1")
	subscriptions.Register(api)

	return router
}
