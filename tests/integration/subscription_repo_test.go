//go:build integration

// Интеграционные тесты репозитория подписок.
// Требуют запущенный PostgreSQL: make compose-up && make test-integration.
package integration

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Dasadno/testtask/internal/config"
	"github.com/Dasadno/testtask/internal/models"
	"github.com/Dasadno/testtask/internal/repository/postgres"
)

func setupRepo(t *testing.T) *postgres.SubscriptionRepo {
	t.Helper()

	port, err := strconv.Atoi(envOr("POSTGRES_PORT", "5432"))
	if err != nil {
		t.Fatalf("invalid POSTGRES_PORT: %v", err)
	}

	cfg := config.PostgresConfig{
		Host:     envOr("POSTGRES_HOST", "localhost"),
		Port:     port,
		User:     envOr("POSTGRES_USER", "postgres"),
		Password: envOr("POSTGRES_PASSWORD", "postgres"),
		Database: envOr("POSTGRES_DB", "subscriptions"),
		SSLMode:  "disable",
	}

	if err := postgres.RunMigrations(cfg.DSN(), "../../env/migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := postgres.NewPool(context.Background(), cfg)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	return postgres.NewSubscriptionRepo(pool)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestSubscriptionRepo_CRUDL(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	end := models.NewMonthYear(2025, time.December)
	sub := models.Subscription{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      userID,
		StartDate:   models.NewMonthYear(2025, time.July),
		EndDate:     &end,
	}

	// Create
	created, err := repo.Create(ctx, sub)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("Create() returned zero id")
	}
	if created.EndDate == nil || created.EndDate.String() != "12-2025" {
		t.Errorf("Create() EndDate = %v, want 12-2025", created.EndDate)
	}
	t.Cleanup(func() { _ = repo.Delete(ctx, created.ID) })

	// GetByID
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ServiceName != sub.ServiceName || got.Price != sub.Price || got.UserID != userID {
		t.Errorf("GetByID() = %+v, want fields of %+v", got, sub)
	}
	if got.StartDate.String() != "07-2025" {
		t.Errorf("GetByID() StartDate = %s, want 07-2025", got.StartDate)
	}

	// Update: меняем цену и убираем дату окончания
	got.Price = 500
	got.EndDate = nil
	updated, err := repo.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Price != 500 || updated.EndDate != nil {
		t.Errorf("Update() = price %d, end %v; want 500, nil", updated.Price, updated.EndDate)
	}

	// List с фильтром по пользователю
	list, err := repo.List(ctx, models.SubscriptionFilter{UserID: &userID})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Errorf("List() = %d items, want exactly the created subscription", len(list))
	}

	// List с фильтром по названию, не совпадающим с созданным
	other := "Netflix"
	list, err = repo.List(ctx, models.SubscriptionFilter{UserID: &userID, ServiceName: &other})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List(Netflix) = %d items, want 0", len(list))
	}

	// Delete
	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.GetByID(ctx, created.ID); !errors.Is(err, models.ErrSubscriptionNotFound) {
		t.Errorf("GetByID() after delete: error = %v, want ErrSubscriptionNotFound", err)
	}
	if err := repo.Delete(ctx, created.ID); !errors.Is(err, models.ErrSubscriptionNotFound) {
		t.Errorf("Delete() twice: error = %v, want ErrSubscriptionNotFound", err)
	}
}

func TestSubscriptionRepo_GetByID_NotFound(t *testing.T) {
	repo := setupRepo(t)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, models.ErrSubscriptionNotFound) {
		t.Errorf("GetByID() error = %v, want ErrSubscriptionNotFound", err)
	}
}
