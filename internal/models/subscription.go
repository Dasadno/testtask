// Package models содержит доменные модели сервиса.
package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrSubscriptionNotFound возвращается, когда подписка с указанным id не существует.
var ErrSubscriptionNotFound = errors.New("subscription not found")

// Subscription — запись об онлайн-подписке пользователя.
type Subscription struct {
	ID          uuid.UUID
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartDate   MonthYear
	EndDate     *MonthYear
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SubscriptionFilter — фильтры и пагинация для списка подписок.
type SubscriptionFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

// CostFilter — период и фильтры для подсчёта суммарной стоимости подписок.
type CostFilter struct {
	From        MonthYear
	To          MonthYear
	UserID      *uuid.UUID
	ServiceName *string
}
