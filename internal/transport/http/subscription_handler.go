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
	subs.GET("/:id", h.get)
	subs.PUT("/:id", h.update)
	subs.DELETE("/:id", h.delete)
}

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
	case errors.Is(err, models.ErrInvalidSubscription):
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
