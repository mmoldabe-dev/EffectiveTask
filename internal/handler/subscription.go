package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mmoldabe-dev/EffectiveTask/docs"
	"github.com/mmoldabe-dev/EffectiveTask/internal/domain"
	"github.com/mmoldabe-dev/EffectiveTask/internal/middleware"
	"github.com/mmoldabe-dev/EffectiveTask/internal/service"
	httpSwagger "github.com/swaggo/http-swagger"
)

type HandlerSubscription struct {
	services service.SubscriptionServiceInterface
	log      *slog.Logger
}

func NewHandlerSubscription(services service.SubscriptionServiceInterface, log *slog.Logger) *HandlerSubscription {
	return &HandlerSubscription{
		services: services,
		log:      log.With(slog.String("component", "delivery/http")),
	}
}

func (h *HandlerSubscription) SetupRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /subscriptions", h.createSubscription)
	mux.HandleFunc("GET /subscriptions/{id}", h.getSubscription)
	mux.HandleFunc("DELETE /subscriptions/{id}", h.deleteSubscription)
	mux.HandleFunc("GET /subscriptions", h.listSubscription)
	mux.HandleFunc("GET /subscriptions/total", h.getTotalCost)
	mux.HandleFunc("PUT /subscriptions/{id}/extend", h.extendSubscription)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	var handler http.Handler = mux
	handler = middleware.JSONMiddleware(handler)
	handler = middleware.LogginMiddleware(h.log)(handler)
	handler = middleware.RecoverMiddleware(h.log)(handler)

	return handler
}


type CreateSubscriptionRequest struct {
	UserID      uuid.UUID `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ServiceName string    `json:"service_name" example:"Spotify Premium"`
	Price       int       `json:"price" example:"500"`
	StartDate   string    `json:"start_date" example:"01-2026"`
	EndDate     *string   `json:"end_date,omitempty" example:"12-2026"`
}

// @Summary  Create subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param input body CreateSubscriptionRequest true Subscription data"
// @Success 201 {object} map[string]int64 "id"
// @Failure 400 {string} string " Validation error"
// @Failure 409 {string} string "Subscription already exists"
// @Router /subscriptions [post]
func (h *HandlerSubscription) createSubscription(w http.ResponseWriter, r *http.Request) {
	var input domain.Subscription

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.Error("failed to decode request body", slog.String("error", err.Error()))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.UserID == uuid.Nil {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(input.ServiceName) == "" || len(input.ServiceName) > 100 {
		http.Error(w, "service_name is required and must be <= 100 characters", http.StatusBadRequest)
		return
	}

	if input.Price < 0 {
		http.Error(w, "price cannot be negative", http.StatusBadRequest)
		return
	}

	if isInvalidDate(input.StartDate) {
		http.Error(w, "invalid start_date format (MM-YYYY)", http.StatusBadRequest)
		return
	}

	if input.EndDate != nil {
		if isInvalidDate(*input.EndDate) {
			http.Error(w, "invalid end_date format (MM-YYYY)", http.StatusBadRequest)
			return
		}

		startDate, errS := time.Parse("01-2006", input.StartDate)
		endDate, errE := time.Parse("01-2006", *input.EndDate)

		if errS != nil || errE != nil {
			http.Error(w, "invalid date values", http.StatusBadRequest)
			return
		}

		if endDate.Before(startDate) {
			http.Error(w, "end_date must be after or equal to start_date", http.StatusBadRequest)
			return
		}
	}

	id, err := h.services.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		h.log.Error("failed to create subscription", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// @Summary Get subscription

// @Tags subscriptions
// @Produce json
// @Param id path int true "Subscription ID"
// @Success 200 {object} domain.Subscription
// @Failure 404 {string} string "Not found"
// @Router /subscriptions/{id} [get]
func (h *HandlerSubscription) getSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := parseID(idStr)
	if err != nil {
		h.log.Error("invalid id parameter", slog.String("id", idStr))
		http.Error(w, "id must be a positive number", http.StatusBadRequest)
		return
	}

	sub, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("failed to get subscription", slog.Int64("id", id), slog.String("error", err.Error()))
		http.Error(w, "subscription not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

// @Summary  Delete subscription
// @Tags subscriptions
// @Produce json
// @Param id path int true 
// @Success 200 {object} map[string]string "status: deleted"
// @Failure 404 {string} string " Not found"
// @Router /subscriptions/{id} [delete]
func (h *HandlerSubscription) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := parseID(idStr)
	if err != nil {
		h.log.Error("invalid id parameter", slog.String("id", idStr))
		http.Error(w, "id must be a positive number", http.StatusBadRequest)
		return
	}

	if err := h.services.Delete(r.Context(), id); err != nil {
		h.log.Error("failed to delete subscription", slog.Int64("id", id), slog.String("error", err.Error()))
		http.Error(w, "subscription not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// @Summary List subscriptions

// @Tags subscriptions
// @Produce json
// @Param user_id query string true "UUID " example(550e8400-e29b-41d4-a716-446655440000)
// @Param service_name query string false 
// @Param limit query int false  example(5)
// @Param offset query int false  example(0)
// @Param min_price query int false  example(1000)
// @Param max_price query int false  example(1300)
// @Success 200 {array} domain.Subscription
// @Failure 400 {string} string 
// @Router /subscriptions [get]
func (h *HandlerSubscription) listSubscription(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	userIdStr := query.Get("user_id")
	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		h.log.Error("invalid user_id", slog.String("val", userIdStr))
		http.Error(w, "invalid user_id (UUID expected)", http.StatusBadRequest)
		return
	}

	limit := 10
	if lStr := query.Get("limit"); lStr != "" {
		if val, err := strconv.Atoi(lStr); err == nil && val > 0 {
			limit = val
		}
	}
	if limit > 200 {
		http.Error(w, "limit must be <= 200", http.StatusBadRequest)
		return
	}

	offset := 0
	if oStr := query.Get("offset"); oStr != "" {
		if val, err := strconv.Atoi(oStr); err == nil && val >= 0 {
			offset = val
		}
	}

	var minPrice, maxPrice int
	if minStr := query.Get("min_price"); minStr != "" {
		minPrice, _ = strconv.Atoi(minStr)
	}
	if maxStr := query.Get("max_price"); maxStr != "" {
		maxPrice, _ = strconv.Atoi(maxStr)
	}
	if minPrice < 0 || maxPrice < 0 {
		http.Error(w, "prices must be non-negative", http.StatusBadRequest)
		return
	}

	filter := domain.SubscriptionFilter{
		UserID:      userID,
		ServiceName: query.Get("service_name"),
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		Limit:       limit,
		Offset:      offset,
	}

	subs, err := h.services.List(r.Context(), userID, filter)
	if err != nil {
		h.log.Error("failed to get list", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}


type TotalCostResponse struct {
	TotalCost int64             `json:"total_cost" example:"6000"`
	Details   []string          `json:"details" example:"Spotify Premium: 6000"`
	Period    map[string]string `json:"period"`
	Warning   string            `json:"warning,omitempty"`
}

// @Summary Total cost
// @Tags subscriptions
// @Produce json
// @Param user_id query string true "UUID" example(550e8400-e29b-41d4-a716-446655440000)
// @Param from query string true "(MM-YYYY)" example(01-2026)
// @Param to query string true "(MM-YYYY)" example(12-2026)
// @Param service_name query string false "(опционально)" example(Spotify Premium)
// @Success 200 {object} TotalCostResponse
// @Failure 400 {string} string 
// @Router /subscriptions/total [get]
func (h *HandlerSubscription) getTotalCost(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	if isInvalidDate(fromStr) || isInvalidDate(toStr) {
		http.Error(w, "invalid date format (MM-YYYY)", http.StatusBadRequest)
		return
	}

	total, details, err := h.services.GetTotalCost(r.Context(), userID, serviceName, fromStr, toStr)
	if err != nil {
		h.log.Error("failed to calculate total cost", slog.String("error", err.Error()))
		http.Error(w, "failed to calculate cost", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"total_cost": total,
		"details":    details,
		"period": map[string]string{
			"from": fromStr,
			"to":   toStr,
		},
	}

	if toStr != "" {
		toDate, parseErr := time.Parse("01-2006", toStr)
		if parseErr == nil {
			now := time.Now()
			currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			if toDate.After(currentMonth) {
				response["warning"] = "Period includes future dates - this is a forecast based on active subscriptions"
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type ExtendInput struct {
	EndDate string `json:"end_date" example:"12-2027"`
	Price   int    `json:"price" example:"600"`
}

// @Summary Extend subscription

// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path int true  example(1)
// @Param input body ExtendInput true
// @Success 200 {object} map[string]string "status: success"
// @Failure 400 {string} string "Неверные данные или дата"
// @Failure 404 {string} string "Подписка не найдена"
// @Router /subscriptions/{id}/extend [put]
func (h *HandlerSubscription) extendSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := parseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	var req ExtendInput

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if isInvalidDate(req.EndDate) {
		http.Error(w, "invalid date format (MM-YYYY)", http.StatusBadRequest)
		return
	}

	if req.Price < 0 {
		http.Error(w, "price cannot be negative", http.StatusBadRequest)
		return
	}

	if err := h.services.Extend(r.Context(), id, req.EndDate, req.Price); err != nil {
		h.log.Error("failed to extend", slog.String("error", err.Error()))
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "subscription not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
