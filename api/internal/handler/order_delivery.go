package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/notify"
)

type updateOrderDeliveryRequest struct {
	DeliveryWindowStart  string `json:"delivery_window_start"`
	DeliveryWindowEnd    string `json:"delivery_window_end"`
	ExpectedDeliveryAt   string `json:"expected_delivery_at"`
	NotifyCustomer       *bool  `json:"notify_customer"`
}

func (s *Server) UpdateOrderDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid order id"))
		return
	}
	if err := s.requireAuth(r); err != nil {
		writeError(w, err)
		return
	}

	var req updateOrderDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	at := strings.TrimSpace(req.ExpectedDeliveryAt)
	if at == "" {
		at = strings.TrimSpace(req.DeliveryWindowStart)
	}
	if at == "" {
		writeError(w, apperror.BadRequest("expected_delivery_at or delivery_window_start is required"))
		return
	}

	start, err := time.Parse(time.RFC3339, at)
	if err != nil {
		writeError(w, apperror.BadRequest("delivery_window_start must be RFC3339"))
		return
	}
	var end time.Time
	if req.DeliveryWindowEnd != "" {
		end, err = time.Parse(time.RFC3339, req.DeliveryWindowEnd)
		if err != nil {
			writeError(w, apperror.BadRequest("delivery_window_end must be RFC3339"))
			return
		}
	} else {
		end = start.Add(30 * time.Minute)
	}
	if end.Before(start) {
		writeError(w, apperror.BadRequest("delivery_window_end must be after start"))
		return
	}

	notifyCustomer := true
	if req.NotifyCustomer != nil {
		notifyCustomer = *req.NotifyCustomer
	}

	existing, err := s.orders.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("order not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	if err := s.authorizeOrder(r, existing.VendorID); err != nil {
		writeError(w, err)
		return
	}

	o, err := s.orders.UpdateDeliverySchedule(r.Context(), s.tenantID, id, start, end, notifyCustomer)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("order not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	if notifyCustomer {
		orderCopy := o
		go notify.DeliverySchedule(context.Background(), s.agentNotifyURL, s.agentNotifySecret, orderCopy)
	}

	writeJSON(w, http.StatusOK, o)
}
