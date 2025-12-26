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
	const op = "service Create"

	// отрицательная цена это странно
	if sub.Price < 0 {
		return 0, fmt.Errorf("op:%s, price must be positive", op)
	}

	// проверяем нет ли уже такой подписки у юзера
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

	s.log.Info("sub created", slog.Int64("id", id))
	return id, nil
}

func (s *SubscriptionService) GetByID(ctx context.Context, id int64) (*domain.Subscription, error) {
	const op = "service GetByID"

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return sub, nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id int64) error {
	const op = "service Delete"

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("%s, %w", op, err)
	}

	return nil
}

func (s *SubscriptionService) List(ctx context.Context, userID uuid.UUID, filter domain.SubscriptionFilter) ([]domain.Subscription, error) {
	const op = "service List"

	// валидация цен, чтоб мин не был больше макса
	if filter.MinPrice > 0 && filter.MaxPrice > 0 && filter.MinPrice > filter.MaxPrice {
		return nil, fmt.Errorf("min price cant be greater than max")
	}

	subs, err := s.repo.List(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return subs, nil
}

func (s *SubscriptionService) GetTotalCost(ctx context.Context, userID uuid.UUID, serviceName string, fromStr, toStr string) (int64, []string, error) {
	const op = "service GetTotalCost"
	layout := "01-2006"

	reqFrom, err := time.Parse(layout, fromStr)
	if err != nil {
		return 0, nil, fmt.Errorf("bad from date format")
	}
	reqTo, err := time.Parse(layout, toStr)
	if err != nil {
		return 0, nil, fmt.Errorf("bad to date format")
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

		// считаем пересечение периодов
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
	const op = "service Extend"

	if !monthYearRegex.MatchString(newEndDateStr) {
		return fmt.Errorf("%s: invalid date format", op)
	}

	if newPrice < 0 {
		return fmt.Errorf("%s: price cant be negative", op)
	}

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: sub not found: %w", op, err)
	}

	startDate, errS := time.Parse("01-2006", sub.StartDate)
	newEndDate, errE := time.Parse("01-2006", newEndDateStr)
	if errS != nil || errE != nil {
		return fmt.Errorf("%s: internal date parse error", op)
	}

	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// нельзя продлевать в прошлое
	if newEndDate.Before(currentMonth) {
		return fmt.Errorf("%s: cant extend to the past", op)
	}

	if newEndDate.Before(startDate) {
		return fmt.Errorf("%s: new end date before start", op)
	}

	if sub.EndDate != nil {
		oldEndDate, _ := time.Parse("01-2006", *sub.EndDate)
		if newEndDate.Before(oldEndDate) || newEndDate.Equal(oldEndDate) {
			return fmt.Errorf("%s: new date must be after old one", op)
		}
	}

	err = s.repo.Extend(ctx, id, newEndDateStr, newPrice)
	if err != nil {
		// логируем если база не обновилась
		s.log.Error("extend update faild", slog.String("op", op), slog.String("err", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
