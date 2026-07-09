package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Dasadno/testtask/internal/models"
)

// SubscriptionService — контракт бизнес-логики, необходимый HTTP-слою.
type SubscriptionService interface {
	Create(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	Get(ctx context.Context, id uuid.UUID) (models.Subscription, error)
	Update(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error)
	TotalCost(ctx context.Context, filter models.CostFilter) (int64, error)
}

// SubscriptionHandler обслуживает HTTP-ручки CRUDL по подпискам.
type SubscriptionHandler struct {
	svc SubscriptionService
	log *slog.Logger
}

// NewSubscriptionHandler создаёт хендлер подписок.
func NewSubscriptionHandler(svc SubscriptionService, log *slog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc, log: log}
}

// Register вешает маршруты подписок на группу.
func (h *SubscriptionHandler) Register(rg *gin.RouterGroup) {
	subs := rg.Group("/subscriptions")
	subs.POST("", h.create)
	subs.GET("", h.list)
	subs.GET("/cost", h.totalCost)
	subs.GET("/:id", h.get)
	subs.PUT("/:id", h.update)
	subs.DELETE("/:id", h.delete)
}

// totalCost godoc
// @Summary     Суммарная стоимость подписок за период
// @Description Для каждой подписки, пересекающейся с периодом, месячная цена умножается на число месяцев пересечения (границы включительно). Подписка без end_date считается бессрочной.
// @Tags        subscriptions
// @Produce     json
// @Param       from         query string true  "Начало периода (MM-YYYY)"
// @Param       to           query string true  "Конец периода (MM-YYYY)"
// @Param       user_id      query string false "Фильтр по ID пользователя (UUID)"
// @Param       service_name query string false "Фильтр по названию сервиса"
// @Success     200 {object} costResponse
// @Failure     400 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /subscriptions/cost [get]
func (h *SubscriptionHandler) totalCost(c *gin.Context) {
	var query costQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.respondBadRequest(c, err)
		return
	}

	filter, err := query.toFilter()
	if err != nil {
		h.respondBadRequest(c, err)
		return
	}

	total, err := h.svc.TotalCost(c.Request.Context(), filter)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, costResponse{From: filter.From, To: filter.To, TotalCost: total})
}

// create godoc
// @Summary     Создать подписку
// @Description Создаёт запись о подписке пользователя. Даты — в формате MM-YYYY.
// @Tags        subscriptions
// @Accept      json
// @Produce     json
// @Param       subscription body subscriptionRequest true "Данные подписки"
// @Success     201 {object} subscriptionResponse
// @Failure     400 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /subscriptions [post]
func (h *SubscriptionHandler) create(c *gin.Context) {
	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondBadRequest(c, err)
		return
	}

	created, err := h.svc.Create(c.Request.Context(), req.toModel())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toSubscriptionResponse(created))
}

// get godoc
// @Summary     Получить подписку
// @Tags        subscriptions
// @Produce     json
// @Param       id path string true "ID подписки (UUID)"
// @Success     200 {object} subscriptionResponse
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /subscriptions/{id} [get]
func (h *SubscriptionHandler) get(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}

	sub, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSubscriptionResponse(sub))
}

// update godoc
// @Summary     Обновить подписку
// @Description Полностью заменяет данные подписки.
// @Tags        subscriptions
// @Accept      json
// @Produce     json
// @Param       id path string true "ID подписки (UUID)"
// @Param       subscription body subscriptionRequest true "Новые данные подписки"
// @Success     200 {object} subscriptionResponse
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /subscriptions/{id} [put]
func (h *SubscriptionHandler) update(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}

	var req subscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondBadRequest(c, err)
		return
	}

	sub := req.toModel()
	sub.ID = id

	updated, err := h.svc.Update(c.Request.Context(), sub)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSubscriptionResponse(updated))
}

// delete godoc
// @Summary     Удалить подписку
// @Tags        subscriptions
// @Param       id path string true "ID подписки (UUID)"
// @Success     204 "No Content"
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /subscriptions/{id} [delete]
func (h *SubscriptionHandler) delete(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// list godoc
// @Summary     Список подписок
// @Description Возвращает подписки с фильтрами и пагинацией (limit по умолчанию 20, максимум 100).
// @Tags        subscriptions
// @Produce     json
// @Param       user_id      query string false "Фильтр по ID пользователя (UUID)"
// @Param       service_name query string false "Фильтр по названию сервиса"
// @Param       limit        query int    false "Размер страницы (1..100)"
// @Param       offset       query int    false "Смещение"
// @Success     200 {object} listSubscriptionsResponse
// @Failure     400 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /subscriptions [get]
func (h *SubscriptionHandler) list(c *gin.Context) {
	var query listSubscriptionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.respondBadRequest(c, err)
		return
	}

	subs, err := h.svc.List(c.Request.Context(), query.toFilter())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toListResponse(subs))
}

// pathID разбирает параметр пути :id; при ошибке отвечает 400 и возвращает ok=false.
func (h *SubscriptionHandler) pathID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "id must be a valid UUID"})
		return uuid.Nil, false
	}
	return id, true
}

// respondBadRequest отвечает 400 с текстом ошибки биндинга/парсинга запроса.
func (h *SubscriptionHandler) respondBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
}

// respondServiceError маппит ошибки бизнес-логики на HTTP-статусы.
func (h *SubscriptionHandler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrSubscriptionNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: models.ErrSubscriptionNotFound.Error()})
	case errors.Is(err, models.ErrInvalidSubscription), errors.Is(err, models.ErrInvalidPeriod):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		// Внутренние детали наружу не отдаём, но логируем с контекстом запроса.
		h.log.ErrorContext(c.Request.Context(), "request failed",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
