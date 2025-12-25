package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/mmoldabe-dev/EffectiveTask/internal/domain"
	"github.com/mmoldabe-dev/EffectiveTask/internal/repository"
)

type SubscriptionServiceInterface interface {
	Create(ctx context.Context, sub domain.Subscription) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Subscription, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, userID uuid.UUID, filter domain.SubscriptionFilter) ([]domain.Subscription, error)
	GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, fromStr, toStr string) (int64, error)
	Extend(ctx context.Context, id int64, newEndDate time.Time) error
}

type SubscriptionService struct {
	repo repository.SubscriptionInterface
	log  *slog.Logger
}

var _ SubscriptionServiceInterface = (*SubscriptionService)(nil)

func NewSubscriptionService(repo repository.SubscriptionInterface, log *slog.Logger) *SubscriptionService {
	return &SubscriptionService{
		repo: repo,
		log:  log.With(slog.String("component", "service")),
	}
}

func (s *SubscriptionService) Create(ctx context.Context, sub domain.Subscription) (int64, error) {
	const op = "service.Subscription.Create"

	if sub.Price < 0 {
		return 0, fmt.Errorf("op:%s, price must be positive", op)
	}

	exists, err := s.repo.Exists(ctx, sub.UserID, sub.ServiceName)
	if err != nil {
		return 0, fmt.Errorf("%s, %w", op, err)
	}
	if exists {
		return 0, fmt.Errorf("%s, such a subscription already exists", op)
	}
	id, err := s.repo.Create(ctx, sub)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	s.log.Info("subscription created successfully", slog.Int64("id", id))
	return id, nil

}

func (s *SubscriptionService) GetByID(ctx context.Context, id int64) (*domain.Subscription, error) {
	const op = "service.Subscription.GetByID"

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return sub, nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id int64) error {
	const op = "service.Subscription.Delete"

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("%s, %w", op, err)
	}

	return nil
}
func (s *SubscriptionService) List(ctx context.Context, userID uuid.UUID, filter domain.SubscriptionFilter) ([]domain.Subscription, error) {
	const op = "service.Subscription.List"

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	subs, err := s.repo.List(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return subs, nil
}
func (s *SubscriptionService) GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, fromStr, toStr string) (int64, error) {
	const op = "service.Subscription.GetTotalCost"

	layout := "01-2006"
	from, err := time.Parse(layout, fromStr)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid 'from' date: %w", op, err)
	}

	to, err := time.Parse(layout, toStr)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid 'to' date: %w", op, err)
	}
	if to.Before(from) {
		return 0, fmt.Errorf("%s: 'to' date cannot be before 'from' date", op)
	}

	total, err := s.repo.GetTotalCost(ctx, userID, serviceName, from, to)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return total, nil
}

func (s *SubscriptionService) Extend(ctx context.Context, id int64, newEndDate time.Time) error {
	const op = "service.Subscription.Extend"

	if newEndDate.Before(time.Now()) {
		return fmt.Errorf("%s: cannot extend to a past date", op)
	}

	err := s.repo.Extend(ctx, id, newEndDate)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
