package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Dasadno/testtask/internal/models"
)

const subscriptionColumns = "id, service_name, price, user_id, start_date, end_date, created_at, updated_at"

// SubscriptionRepo — репозиторий подписок поверх PostgreSQL.
type SubscriptionRepo struct {
	pool *pgxpool.Pool
}

// NewSubscriptionRepo создаёт репозиторий подписок.
func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{pool: pool}
}

// Create сохраняет новую подписку и возвращает её с заполненными id и таймстампами.
func (r *SubscriptionRepo) Create(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	query := `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + subscriptionColumns

	row := r.pool.QueryRow(ctx, query, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate)

	created, err := scanSubscription(row)
	if err != nil {
		return models.Subscription{}, fmt.Errorf("insert subscription: %w", err)
	}

	return created, nil
}

// GetByID возвращает подписку по id или models.ErrSubscriptionNotFound.
func (r *SubscriptionRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + ` FROM subscriptions WHERE id = $1`

	sub, err := scanSubscription(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Subscription{}, models.ErrSubscriptionNotFound
		}
		return models.Subscription{}, fmt.Errorf("select subscription %s: %w", id, err)
	}

	return sub, nil
}

// Update полностью обновляет подписку по id или возвращает models.ErrSubscriptionNotFound.
func (r *SubscriptionRepo) Update(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	query := `
		UPDATE subscriptions
		SET service_name = $2, price = $3, user_id = $4, start_date = $5, end_date = $6, updated_at = now()
		WHERE id = $1
		RETURNING ` + subscriptionColumns

	row := r.pool.QueryRow(ctx, query, sub.ID, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate)

	updated, err := scanSubscription(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Subscription{}, models.ErrSubscriptionNotFound
		}
		return models.Subscription{}, fmt.Errorf("update subscription %s: %w", sub.ID, err)
	}

	return updated, nil
}

// Delete удаляет подписку по id или возвращает models.ErrSubscriptionNotFound.
func (r *SubscriptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete subscription %s: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		return models.ErrSubscriptionNotFound
	}

	return nil
}

// List возвращает подписки с фильтрами по пользователю и названию сервиса.
func (r *SubscriptionRepo) List(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error) {
	var (
		sb   strings.Builder
		args []any
	)

	sb.WriteString(`SELECT ` + subscriptionColumns + ` FROM subscriptions`)

	var conds []string
	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		conds = append(conds, "user_id = $"+strconv.Itoa(len(args)))
	}
	if filter.ServiceName != nil {
		args = append(args, *filter.ServiceName)
		conds = append(conds, "service_name = $"+strconv.Itoa(len(args)))
	}
	if len(conds) > 0 {
		sb.WriteString(" WHERE " + strings.Join(conds, " AND "))
	}

	sb.WriteString(" ORDER BY created_at DESC, id")

	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		sb.WriteString(" LIMIT $" + strconv.Itoa(len(args)))
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		sb.WriteString(" OFFSET $" + strconv.Itoa(len(args)))
	}

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("select subscriptions: %w", err)
	}
	defer rows.Close()

	subs := make([]models.Subscription, 0)
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subscription row: %w", err)
		}
		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscription rows: %w", err)
	}

	return subs, nil
}

func scanSubscription(row pgx.Row) (models.Subscription, error) {
	var sub models.Subscription

	err := row.Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&sub.EndDate,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		return models.Subscription{}, err
	}

	return sub, nil
}
