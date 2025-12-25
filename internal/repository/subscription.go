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
	GetTotalCost(ctx context.Context, userID uuid.UUID, from, to time.Time) (int64, error)
	Exists(ctx context.Context, userID uuid.UUID, serviceName string) (bool, error)
	Extend(ctx context.Context, id int64, newEndDate time.Time) error
}

type SubscriptionRepository struct {
	db  *sql.DB
	log *slog.Logger
}

func NewSubscriptionRepository(db *sql.DB, log *slog.Logger) *SubscriptionRepository {
	return &SubscriptionRepository{
		db:  db,
		log: log.With(slog.String("component", "repository")),
	}
}

// Запись подписки
func (r *SubscriptionRepository) Create(ctx context.Context, sub domain.Subscription) (int64, error) {
	const op = "repository.postgres.Create"
	query := `INSERT INTO subscriptions(service_name, price, user_id, start_date, end_date), VALUES($1, $2, $3, $4,$5)
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

	// Базовый запрос
	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at 
				FROM subscriptions 
				WHERE user_id = $1`

	// список аргументов
	args := []interface{}{userID}
	//некст аргумент
	argID := 2

	//проверка есть ли фильтр на название
	if filter.ServiceName != "" {
		query += fmt.Sprintf(" AND service_name ILIKE $%d", argID)
		args = append(args, "%"+filter.ServiceName+"%")
		argID++
	}

	//лимит
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argID)
		args = append(args, filter.Limit)
		argID++
	}
	//офсет
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argID)
		args = append(args, filter.Offset)
		argID++
	}

	//выполенение запроса
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.log.Error("failed to get list", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()
	//запись данных
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
//подсчета суммарной стоимости всех подписок за период 
func (r *SubscriptionRepository) GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, from, to time.Time) (int64, error) {
	const op = "repository.postgres.GetTotalCost"

	query := `
		SELECT COALESCE(SUM(price), 0) 
		FROM subscriptions 
		WHERE user_id = $1 
		  AND start_date <= $2 
		  AND (end_date IS NULL OR end_date >= $3)`

	args := []interface{}{userID, to, from}
	argID := 4

	if serviceName != "" {
		query += fmt.Sprintf(" AND service_name = $%d", argID)
		args = append(args, serviceName)
	}
	var total int64

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {

		r.log.Error("failed to calculate total cost", slog.String("op:", op), slog.String("error", err.Error()))
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return total, nil

}
//Проверка на exists
func (r *SubscriptionRepository) Exists(ctx context.Context, userID uuid.UUID, serviceName string) (bool, error) {
	const op = "repository.postgres.Exists"
	query := `select exists(
		select 1 form subscriptions where user_id = $1 and service_name =$2 and(end_date IS NULL OR end_date > NOW()) 
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

//Продление подписки
func (r *SubscriptionRepository) Extend(ctx context.Context, id int64, newEndDate time.Time) error {
	const op = "repository.postgres.Extend"

	query := `
		UPDATE subscriptions 
		SET end_date = $1, updated_at = NOW() 
		WHERE id = $2`

	res, err := r.db.ExecContext(ctx, query, newEndDate, id)
	if err != nil {
		r.log.Error("failed to extend subscription", 
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
		return fmt.Errorf("%s: subscription not found", op)
	}

	return nil
}
