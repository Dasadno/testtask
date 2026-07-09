package http

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Dasadno/testtask/internal/models"
)

// subscriptionRequest — тело запроса на создание/полное обновление подписки.
// Обязательные поля объявлены указателями, чтобы отличать «не передано» от нулевого значения.
type subscriptionRequest struct {
	ServiceName string            `json:"service_name" binding:"required" example:"Yandex Plus"`
	Price       *int              `json:"price" binding:"required" example:"400"`
	UserID      string            `json:"user_id" binding:"required,uuid" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   *models.MonthYear `json:"start_date" binding:"required" swaggertype:"string" example:"07-2025"`
	EndDate     *models.MonthYear `json:"end_date" swaggertype:"string" example:"12-2025"`
}

// toModel собирает доменную модель; id заполняется вызывающей стороной при обновлении.
func (r subscriptionRequest) toModel() models.Subscription {
	return models.Subscription{
		ServiceName: r.ServiceName,
		Price:       *r.Price,
		UserID:      uuid.MustParse(r.UserID), // формат проверен binding-тегом uuid
		StartDate:   *r.StartDate,
		EndDate:     r.EndDate,
	}
}

// listSubscriptionsQuery — параметры списка подписок.
type listSubscriptionsQuery struct {
	UserID      string `form:"user_id" binding:"omitempty,uuid"`
	ServiceName string `form:"service_name"`
	Limit       int    `form:"limit" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset" binding:"omitempty,gte=0"`
}

func (q listSubscriptionsQuery) toFilter() models.SubscriptionFilter {
	filter := models.SubscriptionFilter{
		Limit:  q.Limit,
		Offset: q.Offset,
	}
	if q.UserID != "" {
		id := uuid.MustParse(q.UserID) // формат проверен binding-тегом uuid
		filter.UserID = &id
	}
	if q.ServiceName != "" {
		filter.ServiceName = &q.ServiceName
	}
	return filter
}

// costQuery — параметры подсчёта суммарной стоимости.
// Даты разбираются вручную: form-биндинг, в отличие от JSON, не умеет кастомные типы.
type costQuery struct {
	From        string `form:"from" binding:"required"`
	To          string `form:"to" binding:"required"`
	UserID      string `form:"user_id" binding:"omitempty,uuid"`
	ServiceName string `form:"service_name"`
}

func (q costQuery) toFilter() (models.CostFilter, error) {
	from, err := models.ParseMonthYear(q.From)
	if err != nil {
		return models.CostFilter{}, fmt.Errorf("from: %w", err)
	}
	to, err := models.ParseMonthYear(q.To)
	if err != nil {
		return models.CostFilter{}, fmt.Errorf("to: %w", err)
	}

	filter := models.CostFilter{From: from, To: to}
	if q.UserID != "" {
		id := uuid.MustParse(q.UserID) // формат проверен binding-тегом uuid
		filter.UserID = &id
	}
	if q.ServiceName != "" {
		filter.ServiceName = &q.ServiceName
	}
	return filter, nil
}

// costResponse — ответ ручки подсчёта стоимости: эхо периода и итоговая сумма.
type costResponse struct {
	From      models.MonthYear `json:"from" swaggertype:"string" example:"01-2025"`
	To        models.MonthYear `json:"to" swaggertype:"string" example:"06-2025"`
	TotalCost int64            `json:"total_cost" example:"1800"`
}

// subscriptionResponse — представление подписки в API.
type subscriptionResponse struct {
	ID          uuid.UUID         `json:"id" example:"93af4a3e-6f34-4502-a3cf-23dd81c86f1e"`
	ServiceName string            `json:"service_name" example:"Yandex Plus"`
	Price       int               `json:"price" example:"400"`
	UserID      uuid.UUID         `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   models.MonthYear  `json:"start_date" swaggertype:"string" example:"07-2025"`
	EndDate     *models.MonthYear `json:"end_date,omitempty" swaggertype:"string" example:"12-2025"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func toSubscriptionResponse(sub models.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID:          sub.ID,
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID,
		StartDate:   sub.StartDate,
		EndDate:     sub.EndDate,
		CreatedAt:   sub.CreatedAt,
		UpdatedAt:   sub.UpdatedAt,
	}
}

// listSubscriptionsResponse — ответ списка подписок.
type listSubscriptionsResponse struct {
	Items []subscriptionResponse `json:"items"`
}

func toListResponse(subs []models.Subscription) listSubscriptionsResponse {
	items := make([]subscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		items = append(items, toSubscriptionResponse(sub))
	}
	return listSubscriptionsResponse{Items: items}
}

// errorResponse — единый формат ошибок API.
type errorResponse struct {
	Error string `json:"error" example:"subscription not found"`
}
