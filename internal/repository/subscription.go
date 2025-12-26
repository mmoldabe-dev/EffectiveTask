package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/mmoldabe-dev/EffectiveTask/internal/domain"
)

type SubscriptionInterface interface {
	Create(ctx context.Context, sub domain.Subscription) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Subscription, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, userID uuid.UUID, filter domain.SubscriptionFilter) ([]domain.Subscription, error)
	GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, from, to time.Time) ([]domain.Subscription, error)
	Exists(ctx context.Context, userID uuid.UUID, serviceName string) (bool, error)
	Extend(ctx context.Context, id int64, newEndDate string, newPrice int) error
}

type SubscriptionRepository struct {
	db  *sql.DB
	log *slog.Logger
}

var _ SubscriptionInterface = (*SubscriptionRepository)(nil)

func NewSubscriptionRepository(db *sql.DB, log *slog.Logger) *SubscriptionRepository {
	return &SubscriptionRepository{
		db:  db,
		log: log.With(slog.String("component", "repository")),
	}
}

// Запись подписки
func (r *SubscriptionRepository) Create(ctx context.Context, sub domain.Subscription) (int64, error) {
	const op = "repository.postgres.Create"
	query := `INSERT INTO subscriptions(service_name, price, user_id, start_date, end_date)	 VALUES($1, $2, $3, $4,$5)
	RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate).Scan(&id)
	if err != nil {
		r.log.Error("faileed to create subscription", slog.String("op:", op), slog.String("error", err.Error()))
		return 0, err
	}
	return id, nil

}

// Вывести подписку по id
func (r *SubscriptionRepository) GetByID(ctx context.Context, id int64) (*domain.Subscription, error) {
	const op = "repository.postgres.GetByID"
	query := `
	SELECT id, service_name, price,user_id, start_date, end_date,created_at, updated_at from subscriptions 
	WHERE id=$1`

	var sub domain.Subscription

	err := r.db.QueryRowContext(ctx, query, id).Scan(
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
		if err == sql.ErrNoRows {

			return nil, fmt.Errorf("%s: subscription not found: %w", op, err)
		}

		r.log.Error("failed to get subscription",
			slog.String("op", op),
			slog.Int64("id", id),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &sub, nil
}

// удалние подписки
func (r *SubscriptionRepository) Delete(ctx context.Context, id int64) error {
	const op = "repository.postgres.Delete"
	query := `DELETE FROM subscriptions WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.Error("failed to execute delete query",
			slog.String("op", op),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: %w", op, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get rows affected: %w", op, err)
	}
	if rows == 0 {

		return fmt.Errorf("%s: subscription with id %d not found", op, id)
	}
	r.log.Info("subscription deleted successfully", slog.Int64("id", id))
	return nil
}

// фильтр
func (r *SubscriptionRepository) List(ctx context.Context, userID uuid.UUID, filter domain.SubscriptionFilter) ([]domain.Subscription, error) {
	const op = "repository.postgres.List"

	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at 
              FROM subscriptions 
              WHERE user_id = $1`

	args := []interface{}{userID}

	if filter.ServiceName != "" {
		args = append(args, "%"+filter.ServiceName+"%")
		query += fmt.Sprintf(" AND service_name ILIKE $%d", len(args))
	}

	if filter.MinPrice > 0 {
		args = append(args, filter.MinPrice)
		query += fmt.Sprintf(" AND price >= $%d", len(args))
	}

	if filter.MaxPrice > 0 {
		args = append(args, filter.MaxPrice)
		query += fmt.Sprintf(" AND price <= $%d", len(args))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}

	args = append(args, limit)
	query += fmt.Sprintf(" LIMIT $%d", len(args))

	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.log.Error("failed to get list", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var subs []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		err := rows.Scan(
			&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID,
			&sub.StartDate, &sub.EndDate, &sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		subs = append(subs, sub)
	}

	return subs, nil
}

// подсчета суммарной стоимости всех подписок за период
func (r *SubscriptionRepository) GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, from, to time.Time) ([]domain.Subscription, error) {
	const op = "repository.postgres.GetForPeriod"

	query := `
        SELECT price, start_date, end_date 
        FROM subscriptions 
        WHERE user_id = $1 
          AND TO_DATE(start_date, 'MM-YYYY') <= $3
          AND (end_date IS NULL OR TO_DATE(end_date, 'MM-YYYY') >= $2)`

	args := []interface{}{userID, from, to}
	if serviceName != "" {
		query += " AND service_name = $4"
		args = append(args, serviceName)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var subs []domain.Subscription
	for rows.Next() {
		var s domain.Subscription
		if err := rows.Scan(&s.Price, &s.StartDate, &s.EndDate); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

// Проверка на exists
func (r *SubscriptionRepository) Exists(ctx context.Context, userID uuid.UUID, serviceName string) (bool, error) {
	const op = "repository.postgres.Exists"
	query := `select exists(
    select 1 from subscriptions 
    where user_id = $1 
      and service_name = $2 
      and (end_date IS NULL OR TO_DATE(end_date, 'MM-YYYY') >= DATE_TRUNC('month', NOW()))
)`

	var exists bool

	err := r.db.QueryRowContext(ctx, query, userID, serviceName).Scan(&exists)
	if err != nil {
		r.log.Error("failed to check subscription existence",
			slog.String("op", op), slog.String("error", err.Error()),
		)
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

// Продление подписки
func (r *SubscriptionRepository) Extend(ctx context.Context, id int64, newEndDate string, newPrice int) error {
	const op = "repository.postgres.Extend"
	query := `UPDATE subscriptions SET end_date = $1, price = $2, updated_at = NOW() WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, newEndDate, newPrice, id)
	if err != nil {
		r.log.Error("failed to extend subscription", slog.String("op", op), slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get rows affected: %w", op, err)
	}
	if rows == 0 {
		return fmt.Errorf("%s: subscription not found", op)
	}

	return nil
}
