package models

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrInvalidSubscription — базовая ошибка валидации подписки.
// Конкретная причина оборачивается через fmt.Errorf("%w: ...").
var ErrInvalidSubscription = errors.New("invalid subscription")

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
