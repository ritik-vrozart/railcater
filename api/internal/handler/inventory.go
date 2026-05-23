package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

type adjustStockRequest struct {
	Delta  int    `json:"delta"`
	Reason string `json:"reason"`
}

func (s *Server) AdjustStock(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid product id"))
		return
	}

	var req adjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if req.Reason == "" {
		req.Reason = "adjustment"
	}

	p, err := s.inventory.Adjust(r.Context(), s.tenantID, productID, req.Delta, req.Reason)
	var httpErr *apperror.HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, err)
		return
	}
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("product not found"))
		return
	} else if errors.Is(err, apperror.ErrInsufficient) {
		writeError(w, apperror.Unprocessable("insufficient stock for adjustment"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) ListStockMovements(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid product id"))
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	movements, err := s.inventory.ListMovements(r.Context(), s.tenantID, productID, limit)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": movements})
}
