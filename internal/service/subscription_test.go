package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Dasadno/testtask/internal/models"
	"github.com/Dasadno/testtask/internal/service"
)

// repoMock — ручной мок SubscriptionRepository: каждый метод задаётся функцией.
type repoMock struct {
	createFn func(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	getFn    func(ctx context.Context, id uuid.UUID) (models.Subscription, error)
	updateFn func(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	deleteFn func(ctx context.Context, id uuid.UUID) error
	listFn   func(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error)
	costFn   func(ctx context.Context, filter models.CostFilter) (int64, error)
}

func (m *repoMock) Create(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	return m.createFn(ctx, sub)
}

func (m *repoMock) GetByID(ctx context.Context, id uuid.UUID) (models.Subscription, error) {
	return m.getFn(ctx, id)
}

func (m *repoMock) Update(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	return m.updateFn(ctx, sub)
}

func (m *repoMock) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func (m *repoMock) List(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error) {
	return m.listFn(ctx, filter)
}

func (m *repoMock) TotalCost(ctx context.Context, filter models.CostFilter) (int64, error) {
	return m.costFn(ctx, filter)
}

func newService(repo *repoMock) *service.SubscriptionService {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewSubscriptionService(repo, log)
}

func validSubscription() models.Subscription {
	return models.Subscription{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      uuid.New(),
		StartDate:   models.NewMonthYear(2025, time.July),
	}
}

func TestSubscriptionService_Create_Validation(t *testing.T) {
	endBeforeStart := models.NewMonthYear(2025, time.June)

	tests := []struct {
		name   string
		modify func(*models.Subscription)
	}{
		{"empty service name", func(s *models.Subscription) { s.ServiceName = "" }},
		{"negative price", func(s *models.Subscription) { s.Price = -1 }},
		{"nil user id", func(s *models.Subscription) { s.UserID = uuid.Nil }},
		{"zero start date", func(s *models.Subscription) { s.StartDate = models.MonthYear{} }},
		{"end before start", func(s *models.Subscription) { s.EndDate = &endBeforeStart }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &repoMock{
				createFn: func(context.Context, models.Subscription) (models.Subscription, error) {
					t.Fatal("repo must not be called on validation error")
					return models.Subscription{}, nil
				},
			}

			sub := validSubscription()
			tt.modify(&sub)

			_, err := newService(repo).Create(context.Background(), sub)
			if !errors.Is(err, models.ErrInvalidSubscription) {
				t.Errorf("Create() error = %v, want ErrInvalidSubscription", err)
			}
		})
	}
}

func TestSubscriptionService_Create_OK(t *testing.T) {
	sub := validSubscription()
	// Цена 0 и end_date, равный start_date, — валидные граничные значения.
	sub.Price = 0
	end := sub.StartDate
	sub.EndDate = &end

	repo := &repoMock{
		createFn: func(_ context.Context, s models.Subscription) (models.Subscription, error) {
			s.ID = uuid.New()
			return s, nil
		},
	}

	created, err := newService(repo).Create(context.Background(), sub)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() must return subscription with id")
	}
}

func TestSubscriptionService_Get_NotFoundPassthrough(t *testing.T) {
	repo := &repoMock{
		getFn: func(context.Context, uuid.UUID) (models.Subscription, error) {
			return models.Subscription{}, models.ErrSubscriptionNotFound
		},
	}

	_, err := newService(repo).Get(context.Background(), uuid.New())
	if !errors.Is(err, models.ErrSubscriptionNotFound) {
		t.Errorf("Get() error = %v, want ErrSubscriptionNotFound wrapped", err)
	}
}

func TestSubscriptionService_List_NormalizesPagination(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
	}{
		{"defaults", 0, 0, service.DefaultListLimit, 0},
		{"limit above max", 1000, 5, service.MaxListLimit, 5},
		{"negative values", -1, -10, service.DefaultListLimit, 0},
		{"in range untouched", 50, 20, 50, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got models.SubscriptionFilter
			repo := &repoMock{
				listFn: func(_ context.Context, f models.SubscriptionFilter) ([]models.Subscription, error) {
					got = f
					return nil, nil
				},
			}

			_, err := newService(repo).List(context.Background(), models.SubscriptionFilter{
				Limit:  tt.limit,
				Offset: tt.offset,
			})
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if got.Limit != tt.wantLimit || got.Offset != tt.wantOffset {
				t.Errorf("List() filter = limit %d offset %d, want %d/%d",
					got.Limit, got.Offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestSubscriptionService_TotalCost_Validation(t *testing.T) {
	tests := []struct {
		name   string
		filter models.CostFilter
	}{
		{"missing from", models.CostFilter{To: models.NewMonthYear(2025, time.June)}},
		{"missing to", models.CostFilter{From: models.NewMonthYear(2025, time.January)}},
		{"to before from", models.CostFilter{
			From: models.NewMonthYear(2025, time.June),
			To:   models.NewMonthYear(2025, time.January),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &repoMock{
				costFn: func(context.Context, models.CostFilter) (int64, error) {
					t.Fatal("repo must not be called on validation error")
					return 0, nil
				},
			}

			_, err := newService(repo).TotalCost(context.Background(), tt.filter)
			if !errors.Is(err, models.ErrInvalidPeriod) {
				t.Errorf("TotalCost() error = %v, want ErrInvalidPeriod", err)
			}
		})
	}
}

func TestSubscriptionService_TotalCost_OK(t *testing.T) {
	// Период из одного месяца (from == to) валиден.
	filter := models.CostFilter{
		From: models.NewMonthYear(2025, time.July),
		To:   models.NewMonthYear(2025, time.July),
	}

	repo := &repoMock{
		costFn: func(_ context.Context, f models.CostFilter) (int64, error) {
			if f != filter {
				t.Errorf("repo received %+v, want %+v", f, filter)
			}
			return 1800, nil
		},
	}

	total, err := newService(repo).TotalCost(context.Background(), filter)
	if err != nil {
		t.Fatalf("TotalCost() error = %v", err)
	}
	if total != 1800 {
		t.Errorf("TotalCost() = %d, want 1800", total)
	}
}

func TestSubscriptionService_Delete_WrapsRepoError(t *testing.T) {
	repoErr := errors.New("connection lost")
	repo := &repoMock{
		deleteFn: func(context.Context, uuid.UUID) error { return repoErr },
	}

	err := newService(repo).Delete(context.Background(), uuid.New())
	if !errors.Is(err, repoErr) {
		t.Errorf("Delete() error = %v, want wrapped repo error", err)
	}
}
