package http

import (
	"time"

	"github.com/google/uuid"

	"github.com/Dasadno/testtask/internal/models"
)

// subscriptionRequest — тело запроса на создание/полное обновление подписки.
// Обязательные поля объявлены указателями, чтобы отличать «не передано» от нулевого значения.
type subscriptionRequest struct {
	ServiceName string            `json:"service_name" binding:"required"`
	Price       *int              `json:"price" binding:"required"`
	UserID      string            `json:"user_id" binding:"required,uuid"`
	StartDate   *models.MonthYear `json:"start_date" binding:"required"`
	EndDate     *models.MonthYear `json:"end_date"`
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

// subscriptionResponse — представление подписки в API.
type subscriptionResponse struct {
	ID          uuid.UUID         `json:"id"`
	ServiceName string            `json:"service_name"`
	Price       int               `json:"price"`
	UserID      uuid.UUID         `json:"user_id"`
	StartDate   models.MonthYear  `json:"start_date"`
	EndDate     *models.MonthYear `json:"end_date,omitempty"`
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
	Error string `json:"error"`
}
