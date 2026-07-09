// Package logger настраивает slog-логгер в зависимости от окружения.
package logger

import (
	"log/slog"
	"os"

	"github.com/Dasadno/testtask/internal/config"
)

// New возвращает логгер: текстовый с уровнем Debug для локальной разработки,
// JSON с уровнем Info для остальных окружений.
func New(env string) *slog.Logger {
	var handler slog.Handler

	switch env {
	case config.EnvLocal:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	default:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	return slog.New(handler)
}
