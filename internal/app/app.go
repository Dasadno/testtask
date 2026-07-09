// Package app собирает зависимости сервиса и управляет его жизненным циклом.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dasadno/testtask/internal/config"
	"github.com/Dasadno/testtask/internal/logger"
	"github.com/Dasadno/testtask/internal/repository/postgres"
	transport "github.com/Dasadno/testtask/internal/transport/http"
)

const dbConnectTimeout = 10 * time.Second

// Run запускает сервис и блокируется до получения SIGINT/SIGTERM,
// после чего корректно останавливает HTTP-сервер.
func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Env)
	log.Info("starting service", slog.String("env", cfg.Env), slog.Int("port", cfg.HTTP.Port))

	if err := postgres.RunMigrations(cfg.Postgres.DSN(), cfg.Postgres.MigrationsPath); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Info("migrations applied")

	connectCtx, cancelConnect := context.WithTimeout(context.Background(), dbConnectTimeout)
	defer cancelConnect()

	pool, err := postgres.NewPool(connectCtx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()
	log.Info("connected to postgres")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      transport.NewRouter(log, cfg.Env),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	log.Info("service stopped")
	return nil
}
