// Package service содержит бизнес-логику сервиса подписок.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Dasadno/testtask/internal/models"
)

// Пагинация списка по умолчанию и её верхняя граница.
const (
	DefaultListLimit = 20
	MaxListLimit     = 100
)

// SubscriptionRepository — контракт слоя хранения, необходимый сервису.
// Интерфейс объявлен на стороне потребителя; реализация — repository/postgres.
type SubscriptionRepository interface {
	Create(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Subscription, error)
	Update(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error)
	TotalCost(ctx context.Context, filter models.CostFilter) (int64, error)
}

// SubscriptionService реализует бизнес-логику работы с подписками.
type SubscriptionService struct {
	repo SubscriptionRepository
	log  *slog.Logger
}

// NewSubscriptionService создаёт сервис подписок.
func NewSubscriptionService(repo SubscriptionRepository, log *slog.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, log: log}
}

// Create валидирует и сохраняет новую подписку.
func (s *SubscriptionService) Create(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	if err := sub.Validate(); err != nil {
		return models.Subscription{}, err
	}

	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		return models.Subscription{}, fmt.Errorf("create subscription: %w", err)
	}

	s.log.InfoContext(ctx, "subscription created",
		slog.String("id", created.ID.String()),
		slog.String("user_id", created.UserID.String()),
		slog.String("service_name", created.ServiceName),
	)

	return created, nil
}

// Get возвращает подписку по id.
func (s *SubscriptionService) Get(ctx context.Context, id uuid.UUID) (models.Subscription, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return models.Subscription{}, fmt.Errorf("get subscription: %w", err)
	}
	return sub, nil
}

// Update валидирует и полностью обновляет подписку.
func (s *SubscriptionService) Update(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	if err := sub.Validate(); err != nil {
		return models.Subscription{}, err
	}

	updated, err := s.repo.Update(ctx, sub)
	if err != nil {
		return models.Subscription{}, fmt.Errorf("update subscription: %w", err)
	}

	s.log.InfoContext(ctx, "subscription updated", slog.String("id", updated.ID.String()))

	return updated, nil
}

// Delete удаляет подписку по id.
func (s *SubscriptionService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}

	s.log.InfoContext(ctx, "subscription deleted", slog.String("id", id.String()))

	return nil
}

// List возвращает подписки по фильтру, нормализуя пагинацию.
func (s *SubscriptionService) List(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error) {
	if filter.Limit <= 0 {
		filter.Limit = DefaultListLimit
	}
	if filter.Limit > MaxListLimit {
		filter.Limit = MaxListLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	subs, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}

	return subs, nil
}

// TotalCost считает суммарную стоимость подписок за период с учётом фильтров.
func (s *SubscriptionService) TotalCost(ctx context.Context, filter models.CostFilter) (int64, error) {
	if err := filter.Validate(); err != nil {
		return 0, err
	}

	total, err := s.repo.TotalCost(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("total cost: %w", err)
	}

	s.log.InfoContext(ctx, "total cost calculated",
		slog.String("from", filter.From.String()),
		slog.String("to", filter.To.String()),
		slog.Int64("total", total),
	)

	return total, nil
}
