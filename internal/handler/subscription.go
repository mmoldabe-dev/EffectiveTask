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
	// накидываем мидлвары
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

// @Summary Create subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param input body CreateSubscriptionRequest true "Subscription info"
// @Success 201 {object} map[string]int64
// @Failure 400 {string} string
// @Failure 409 {string} string
// @Router /subscriptions [post]
func (h *HandlerSubscription) createSubscription(w http.ResponseWriter, r *http.Request) {
	var input domain.Subscription
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.log.Error("body decode fail", slog.String("err", err.Error()))
		http.Error(w, "invalid request body", 400)
		return
	}

	// валидация входных данных
	if input.UserID == uuid.Nil {
		http.Error(w, "user_id is required", 400)
		return
	}

	if strings.TrimSpace(input.ServiceName) == "" || len(input.ServiceName) > 100 {
		http.Error(w, "service_name too long or empty", 400)
		return
	}

	if input.Price < 0 {
		http.Error(w, "price cant be negative", 400)
		return
	}

	if isInvalidDate(input.StartDate) {
		http.Error(w, "bad start_date (MM-YYYY)", 400)
		return
	}

	if input.EndDate != nil {
		if isInvalidDate(*input.EndDate) {
			http.Error(w, "bad end_date", 400)
			return
		}

		sDate, _ := time.Parse("01-2006", input.StartDate)
		eDate, _ := time.Parse("01-2006", *input.EndDate)

		if eDate.Before(sDate) {
			http.Error(w, "end date before start date", 400)
			return
		}
	}

	id, err := h.services.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionExists) {
			http.Error(w, err.Error(), 409)
			return
		}
		h.log.Error("create failed", slog.String("err", err.Error()))
		http.Error(w, "internal error", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// @Summary Get subscription details
// @Tags subscriptions
// @Produce json
// @Param id path int true "Subscription ID"
// @Success 200 {object} domain.Subscription
// @Failure 404 {string} string
// @Router /subscriptions/{id} [get]
func (h *HandlerSubscription) getSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := parseID(idStr)
	if err != nil {
		h.log.Error("bad id param", slog.String("id", idStr))
		http.Error(w, "id must be positive", 400)
		return
	}

	sub, err := h.services.GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("get sub fail", slog.Int64("id", id), slog.String("err", err.Error()))
		http.Error(w, "sub not found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

// @Summary Delete subscription
// @Tags subscriptions
// @Param id path int true "Subscription ID"
// @Success 200 {object} map[string]string
// @Failure 404 {string} string
// @Router /subscriptions/{id} [delete]
func (h *HandlerSubscription) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := parseID(idStr)
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}

	if err := h.services.Delete(r.Context(), id); err != nil {
		h.log.Error("delete fail", slog.Int64("id", id), slog.String("error", err.Error()))
		http.Error(w, "not found", 404)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// @Summary List subscriptions
// @Tags subscriptions
// @Produce json
// @Param user_id query string true "User UUID"
// @Param service_name query string false "Service filter"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param min_price query int false "Min price"
// @Param max_price query int false "Max price"
// @Success 200 {array} domain.Subscription
// @Failure 400 {string} string
// @Router /subscriptions [get]
func (h *HandlerSubscription) listSubscription(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	uIDStr := q.Get("user_id")
	uID, err := uuid.Parse(uIDStr)
	if err != nil {
		h.log.Error("bad user_id", slog.String("val", uIDStr))
		http.Error(w, "invalid user_id", 400)
		return
	}

	limit := 10
	if l := q.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	
	if limit > 200 {
		http.Error(w, "limit too big", 400)
		return
	}

	offset, _ := strconv.Atoi(q.Get("offset"))
	minP, _ := strconv.Atoi(q.Get("min_price"))
	maxP, _ := strconv.Atoi(q.Get("max_price"))

	filter := domain.SubscriptionFilter{
		UserID:      uID,
		ServiceName: q.Get("service_name"),
		MinPrice:    minP, MaxPrice: maxP,
		Limit: limit, Offset: offset,
	}

	subs, err := h.services.List(r.Context(), uID, filter)
	if err != nil {
		h.log.Error("list fail", slog.String("error", err.Error()))
		http.Error(w, "internal error", 500)
		return
	}

	json.NewEncoder(w).Encode(subs)
}

type TotalCostResponse struct {
	TotalCost int64             `json:"total_cost" example:"6000"`
	Details   []string          `json:"details" example:"Spotify Premium: 6000"`
	Period    map[string]string `json:"period"`
	Warning   string            `json:"warning,omitempty"`
}

// @Summary Calculate total cost
// @Tags subscriptions
// @Produce json
// @Param user_id query string true "User UUID"
// @Param from query string true "Start date (MM-YYYY)"
// @Param to query string true "End date (MM-YYYY)"
// @Param service_name query string false "Service filter(не обязатльно)"
// @Success 200 {object} TotalCostResponse
// @Failure 400 {string} string
// @Router /subscriptions/total [get]
func (h *HandlerSubscription) getTotalCost(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	uIDStr := params.Get("user_id")
	fromStr := params.Get("from")
	toStr := params.Get("to")

	uID, err := uuid.Parse(uIDStr)
	if err != nil {
		http.Error(w, "bad user_id", 400)
		return
	}

	if isInvalidDate(fromStr) || isInvalidDate(toStr) {
		http.Error(w, "invalid date format", 400)
		return
	}

	total, details, err := h.services.GetTotalCost(r.Context(), uID, params.Get("service_name"), fromStr, toStr)
	if err != nil {
		h.log.Error("cost calc faild", slog.String("err", err.Error()))
		http.Error(w, "failed to calculate cost", 400)
		return
	}

	resp := map[string]interface{}{
		"total_cost": total,
		"details":    details,
		"period": map[string]string{
			"from": fromStr, "to": toStr,
		},
	}

	// чекаем если дата в будущем, кидаем ворнинг
	if toStr != "" {
		if tDate, e := time.Parse("01-2006", toStr); e == nil {
			now := time.Now()
			curr := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			if tDate.After(curr) {
				resp["warning"] = "Period includes future dates - forecast based on active subs"
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type ExtendInput struct {
	EndDate string `json:"end_date" example:"12-2027"`
	Price   int    `json:"price" example:"600"`
}

// @Summary Extend subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path int true "Subscription ID"
// @Param input body ExtendInput true "New data"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Router /subscriptions/{id}/extend [put]
func (h *HandlerSubscription) extendSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}

	var req ExtendInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	if isInvalidDate(req.EndDate) || req.Price < 0 {
		http.Error(w, "invalid data", 400)
		return
	}

	if err := h.services.Extend(r.Context(), id, req.EndDate, req.Price); err != nil {
		h.log.Error("extend fail", slog.String("err", err.Error()))
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "not found", 404)
			return
		}
		http.Error(w, err.Error(), 400)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}