package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

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
	mux.HandleFunc("GET /subscriptions/total", h.listSubscription)
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

	var filter domain.SubscriptionFilter

	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		h.log.Error("failed to decode request body", slog.String("error", err.Error()))
		http.Error(w, "invalid request body", http.StatusBadRequest)
	}

	subs, err := h.services.List(r.Context(), filter.UserID, filter)
	if err != nil {
		h.log.Error("failed to  get List subscriptions ", slog.String("error", err.Error()))
		http.Error(w, "error", http.StatusNotFound)
	}

	w.Header().Set("Content-Type", "applictaion/JSON")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(subs); err != nil {
		h.log.Error("failed to encode response", slog.String("error", err.Error()))
	}
}
