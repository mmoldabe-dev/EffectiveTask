package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/mmoldabe-dev/EffectiveTask/internal/domain"
	"github.com/mmoldabe-dev/EffectiveTask/internal/repository"
)

var ErrSubscriptionExists = errors.New("subscription already exists")

type SubscriptionServiceInterface interface {
	Create(ctx context.Context, sub domain.Subscription) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.Subscription, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, userID uuid.UUID, filter domain.SubscriptionFilter) ([]domain.Subscription, error)
	GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, fromStr, toStr string) (int64, []string, error)
	Extend(ctx context.Context, id int64, newEndDateStr string, newPrice int) error
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
		return 0, ErrSubscriptionExists
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

	if filter.MinPrice > 0 && filter.MaxPrice > 0 && filter.MinPrice > filter.MaxPrice {
		return nil, fmt.Errorf("minimum price cannot be greater than maximum price")
	}

	subs, err := s.repo.List(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return subs, nil
}
func (s *SubscriptionService) GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, fromStr, toStr string) (int64, []string, error) {
	const op = "service.Subscription.GetTotalCost"
	layout := "01-2006"

	reqFrom, err := time.Parse(layout, fromStr)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid from date")
	}
	reqTo, err := time.Parse(layout, toStr)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid to date")
	}

	subs, err := s.repo.GetTotalCost(ctx, userID, serviceName, reqFrom, reqTo)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", op, err)
	}

	var totalCost int64
	var details []string
	for _, sub := range subs {
		subStart, _ := time.Parse(layout, sub.StartDate)

		var subEnd time.Time
		if sub.EndDate != nil {
			subEnd, _ = time.Parse(layout, *sub.EndDate)
		} else {
			subEnd = reqTo
		}

		intersectStart := maxDate(reqFrom, subStart)
		intersectEnd := minDate(reqTo, subEnd)

		months := countMonths(intersectStart, intersectEnd)

		if months > 0 {
			cost := int64(sub.Price) * int64(months)
			totalCost += cost

			details = append(details, fmt.Sprintf("%s: %d", sub.ServiceName, cost))
		}
	}

	return totalCost, details, nil
}

var monthYearRegex = regexp.MustCompile(`^(0[1-9]|1[0-2])-\d{4}$`)

func (s *SubscriptionService) Extend(ctx context.Context, id int64, newEndDateStr string, newPrice int) error {
	const op = "service.Subscription.Extend"

	if !monthYearRegex.MatchString(newEndDateStr) {
		return fmt.Errorf("%s: invalid date format", op)
	}

	if newPrice < 0 {
		return fmt.Errorf("%s: price cannot be negative", op)
	}

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: subscription not found: %w", op, err)
	}

	startDate, errS := time.Parse("01-2006", sub.StartDate)
	newEndDate, errE := time.Parse("01-2006", newEndDateStr)
	if errS != nil || errE != nil {
		return fmt.Errorf("%s: internal date parse error", op)
	}

	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if newEndDate.Before(currentMonth) {
		return fmt.Errorf("%s: cannot extend to a past date", op)
	}

	if newEndDate.Before(startDate) {
		return fmt.Errorf("%s: new end date cannot be before start date", op)
	}

	if sub.EndDate != nil {
		oldEndDate, _ := time.Parse("01-2006", *sub.EndDate)
		if newEndDate.Before(oldEndDate) || newEndDate.Equal(oldEndDate) {
			return fmt.Errorf("%s: new end date must be after current end date", op)
		}
	}

	err = s.repo.Extend(ctx, id, newEndDateStr, newPrice)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
