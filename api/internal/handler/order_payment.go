package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

type updateOrderPaymentRequest struct {
	PaymentStatus  string  `json:"payment_status"`
	PaymentMethod  *string `json:"payment_method"`
}

func (s *Server) UpdateOrderPayment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid order id"))
		return
	}
	if err := s.requireAuth(r); err != nil {
		writeError(w, err)
		return
	}

	var req updateOrderPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	status := strings.TrimSpace(strings.ToLower(req.PaymentStatus))
	if status != "pending" && status != "paid" {
		writeError(w, apperror.BadRequest("payment_status must be pending or paid"))
		return
	}

	var method *string
	if req.PaymentMethod != nil {
		m := strings.TrimSpace(strings.ToLower(*req.PaymentMethod))
		if m != "" {
			if m != "cash" && m != "upi" {
				writeError(w, apperror.BadRequest("payment_method must be cash or upi"))
				return
			}
			method = &m
		}
	}
	if status == "paid" && method == nil {
		writeError(w, apperror.BadRequest("payment_method (cash or upi) is required when marking paid"))
		return
	}
	if status == "pending" {
		method = nil
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

	o, err := s.orders.UpdatePayment(r.Context(), s.tenantID, id, status, method)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("order not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	writeJSON(w, http.StatusOK, o)
}
