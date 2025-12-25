package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mmoldabe-dev/EffectiveTask/internal/domain"
	"github.com/mmoldabe-dev/EffectiveTask/internal/service"
)

type HandlerSubcription struct {
	services service.SubscriptionServiceInterface
	log      *slog.Logger
}

func NewHandlerSubcription(services service.SubscriptionServiceInterface, log *slog.Logger) *HandlerSubcription {
	return &HandlerSubcription{
		services: services,
		log:      log.With(slog.String("component", "delivery/http")),
	}
}

func (h *HandlerSubcription) SetupRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /subscriptions", h.createSubscription)
	mux.HandleFunc("GET /subscriptions/{id}", h.getSubscription)
	mux.HandleFunc("DELETE /subscriptions/{id}", h.deleteSubscription)
	mux.HandleFunc("GET /subscriptions", h.listSubscription)
	mux.HandleFunc("GET /subscriptions/total", h.getTotalCost)
	mux.HandleFunc("GET /subscriptions/{id}/extend", h.extendSubscription)

	return mux
}

func (h *HandlerSubcription) createSubscription(w http.ResponseWriter, r *http.Request) {
	var input domain.Subscription

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.Error("failed to decode request body", slog.String("error", err.Error()))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id, err := h.services.Create(r.Context(), input)
	if err != nil {
		h.log.Error("failed to create subscription", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (h *HandlerSubcription) getSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Error("invalid id parameter", slog.String("id", idStr))
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	sub, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("failed to get subscription", slog.Int64("id", id), slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sub); err != nil {
		h.log.Error("failed to encode response", slog.String("error", err.Error()))
	}

}

func (h *HandlerSubcription) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Error("invalid id parameter", slog.String("id", idStr))
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	if err := h.services.Delete(r.Context(), id); err != nil {
		h.log.Error("failed to delete subscription", slog.Int64("id", id), slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

}
func (h *HandlerSubcription) listSubscription(w http.ResponseWriter, r *http.Request) {
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
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(subs)
}
func (h *HandlerSubcription) getTotalCost(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	total, err := h.services.GetTotalCost(r.Context(), userID, serviceName, fromStr, toStr)
	if err != nil {
		h.log.Error("failed to calculate total cost", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_cost": total,
		"period": map[string]string{
			"from": fromStr,
			"to":   toStr,
		},
	})
}
func (h *HandlerSubcription) extendSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Error("invalid id parameter", slog.String("id", idStr))
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	dateStr := r.URL.Query().Get("new_date")
	date, err := time.Parse("01-2006", dateStr)
	if err != nil {
		h.log.Error("invalid date parameter", slog.String("date", dateStr))
		http.Error(w, "invalid date parameter", http.StatusBadRequest)
		return
	}

	if err := h.services.Extend(ctx, id, date); err != nil {
		h.log.Error("failed to extend subscription", slog.Any("error", err))
		http.Error(w, "subscription not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
