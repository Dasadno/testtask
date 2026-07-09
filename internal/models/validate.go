package models

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrInvalidSubscription — базовая ошибка валидации подписки.
// Конкретная причина оборачивается через fmt.Errorf("%w: ...").
var ErrInvalidSubscription = errors.New("invalid subscription")

// ErrInvalidPeriod — базовая ошибка валидации периода подсчёта стоимости.
var ErrInvalidPeriod = errors.New("invalid period")

// Validate проверяет бизнес-правила подписки.
func (s Subscription) Validate() error {
	if s.ServiceName == "" {
		return fmt.Errorf("%w: service_name must not be empty", ErrInvalidSubscription)
	}
	if s.Price < 0 {
		return fmt.Errorf("%w: price must be non-negative", ErrInvalidSubscription)
	}
	if s.UserID == uuid.Nil {
		return fmt.Errorf("%w: user_id must not be empty", ErrInvalidSubscription)
	}
	if s.StartDate.IsZero() {
		return fmt.Errorf("%w: start_date must be set", ErrInvalidSubscription)
	}
	if s.EndDate != nil && s.EndDate.Before(s.StartDate.Time) {
		return fmt.Errorf("%w: end_date must not be before start_date", ErrInvalidSubscription)
	}
	return nil
}

// Validate проверяет корректность периода подсчёта стоимости.
func (f CostFilter) Validate() error {
	if f.From.IsZero() {
		return fmt.Errorf("%w: from must be set", ErrInvalidPeriod)
	}
	if f.To.IsZero() {
		return fmt.Errorf("%w: to must be set", ErrInvalidPeriod)
	}
	if f.To.Before(f.From.Time) {
		return fmt.Errorf("%w: to must not be before from", ErrInvalidPeriod)
	}
	return nil
}
