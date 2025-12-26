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
	"github.com/mmoldabe-dev/EffectiveTask/internal/domain"
	"github.com/mmoldabe-dev/EffectiveTask/internal/middleware"
	"github.com/mmoldabe-dev/EffectiveTask/internal/service"
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

	var handler http.Handler = mux
	handler = middleware.JSONMiddleware(handler)
	handler = middleware.LogginMiddleware(h.log)(handler)
	handler = middleware.RecoverMiddleware(h.log)(handler)

	return handler
}

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

	total, err := h.services.GetTotalCost(r.Context(), userID, serviceName, fromStr, toStr)
	if err != nil {
		h.log.Error("failed to calculate total cost", slog.String("error", err.Error()))
		http.Error(w, "failed to calculate cost", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"total_cost": total,
		"period": map[string]string{
			"from": fromStr,
			"to":   toStr,
		},
	}

	// Добавляем предупреждение, если период включает будущие даты
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

func (h *HandlerSubscription) extendSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := parseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	var req struct {
		EndDate string `json:"end_date"`
		Price   int    `json:"price"`
	}

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
