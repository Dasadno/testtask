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

func TestSubscriptionRepo_TotalCost(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()
	// Уникальный пользователь изолирует тест от чужих данных в таблице.
	userID := uuid.New()

	create := func(name string, price int, start models.MonthYear, end *models.MonthYear) {
		t.Helper()
		created, err := repo.Create(ctx, models.Subscription{
			ServiceName: name,
			Price:       price,
			UserID:      userID,
			StartDate:   start,
			EndDate:     end,
		})
		if err != nil {
			t.Fatalf("create fixture %s: %v", name, err)
		}
		t.Cleanup(func() { _ = repo.Delete(ctx, created.ID) })
	}

	end := func(y int, m time.Month) *models.MonthYear {
		v := models.NewMonthYear(y, m)
		return &v
	}

	// Сценарий из дизайн-дока + подписки, не попадающие в период.
	create("Yandex Plus", 400, models.NewMonthYear(2025, time.February), end(2025, time.April)) // 3 мес × 400
	create("Netflix", 300, models.NewMonthYear(2025, time.May), nil)                            // бессрочная: 05,06 → 2 × 300
	create("Spotify", 200, models.NewMonthYear(2024, time.January), end(2024, time.December))   // закончилась до периода
	create("Ivi", 100, models.NewMonthYear(2025, time.July), nil)                               // начнётся после периода

	from := models.NewMonthYear(2025, time.January)
	to := models.NewMonthYear(2025, time.June)

	tests := []struct {
		name    string
		filter  models.CostFilter
		want    int64
	}{
		{
			name:   "period sums overlap months only",
			filter: models.CostFilter{From: from, To: to, UserID: &userID},
			want:   3*400 + 2*300, // 1800
		},
		{
			name: "filter by service name",
			filter: models.CostFilter{
				From: from, To: to, UserID: &userID,
				ServiceName: strPtr("Netflix"),
			},
			want: 600,
		},
		{
			name:   "single month period",
			filter: models.CostFilter{From: models.NewMonthYear(2025, time.March), To: models.NewMonthYear(2025, time.March), UserID: &userID},
			want:   400, // только Yandex Plus активна в марте
		},
		{
			name: "wide period counts unbounded subscription till period end",
			filter: models.CostFilter{
				From: models.NewMonthYear(2025, time.January), To: models.NewMonthYear(2025, time.December),
				UserID: &userID,
			},
			// Yandex Plus: 3×400; Netflix: 05..12 = 8×300; Ivi: 07..12 = 6×100
			want: 1200 + 2400 + 600,
		},
		{
			name:   "no matches returns zero",
			filter: models.CostFilter{From: models.NewMonthYear(2020, time.January), To: models.NewMonthYear(2020, time.December), UserID: &userID},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.TotalCost(ctx, tt.filter)
			if err != nil {
				t.Fatalf("TotalCost() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("TotalCost() = %d, want %d", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestSubscriptionRepo_GetByID_NotFound(t *testing.T) {
	repo := setupRepo(t)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, models.ErrSubscriptionNotFound) {
		t.Errorf("GetByID() error = %v, want ErrSubscriptionNotFound", err)
	}
}
