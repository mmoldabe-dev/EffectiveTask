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
	GetTotalCost(ctx context.Context, userID uuid.UUID, from, to time.Time) (float64, error)
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
//Вывести подписку по id
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
